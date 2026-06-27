// Package transcript reads the tail of an agent's session transcript (JSONL) so
// the LLM reprompt builder has recent context. Formats differ across agents
// (Claude Code vs Codex), so extraction is deliberately schema-loose: it pulls
// role and text from whatever common fields are present and ignores the rest.
package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxMsgChars caps how much of each message is kept, so a giant tool result
// doesn't blow up the context handed to the model.
const maxMsgChars = 600

// Tail returns up to k of the most recent messages from the newest transcript
// file matching glob, rendered as "role: text" lines. The glob may start with
// "~" and may contain a single "**" for recursive matching (Codex layout).
func Tail(glob string, k int) (string, error) {
	if k <= 0 {
		k = 20
	}
	path, err := newest(glob)
	if err != nil {
		return "", err
	}
	lines, err := lastLines(path, k)
	if err != nil {
		return "", err
	}
	var out []string
	for _, line := range lines {
		role, text := extract(line)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if len(text) > maxMsgChars {
			text = text[:maxMsgChars] + "…"
		}
		if role == "" {
			role = "msg"
		}
		out = append(out, role+": "+text)
	}
	return strings.Join(out, "\n"), nil
}

// newest returns the most recently modified file matching glob.
func newest(glob string) (string, error) {
	matches, err := expandGlob(glob)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no transcript files match %q", glob)
	}
	var best string
	var bestMod int64 = -1
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil || fi.IsDir() {
			continue
		}
		if mt := fi.ModTime().UnixNano(); mt > bestMod {
			bestMod, best = mt, m
		}
	}
	if best == "" {
		return "", fmt.Errorf("no readable transcript files match %q", glob)
	}
	return best, nil
}

// expandGlob expands "~" and supports a single "**" segment (filepath.Glob does
// not). With "**" it walks the base directory and matches the trailing pattern
// against file names.
func expandGlob(glob string) ([]string, error) {
	glob = expandHome(glob)
	if !strings.Contains(glob, "**") {
		return filepath.Glob(glob)
	}
	idx := strings.Index(glob, "**")
	base := filepath.Dir(strings.TrimRight(glob[:idx], `/\`))
	suffix := strings.TrimLeft(glob[idx+2:], `/\`)
	var matches []string
	_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if ok, _ := filepath.Match(suffix, d.Name()); ok {
			matches = append(matches, p)
		}
		return nil
	})
	return matches, nil
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// lastLines returns the final k non-empty lines of a file.
func lastLines(path string, k int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // transcript lines can be long
	ring := make([]string, 0, k)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if len(ring) == k {
			ring = ring[1:]
		}
		ring = append(ring, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return ring, nil
}

// extract pulls a role and text from one JSONL message, tolerating both the
// Claude Code shape ({type, message:{role, content:[{type:text,text}]}}) and
// flatter shapes ({role, content}).
func extract(line string) (role, text string) {
	var m map[string]any
	if json.Unmarshal([]byte(line), &m) != nil {
		return "", ""
	}
	role = firstString(m, "role", "type")
	if msg, ok := m["message"].(map[string]any); ok {
		if r := firstString(msg, "role"); r != "" {
			role = r
		}
		text = collectText(msg["content"])
	}
	if text == "" {
		text = collectText(m["content"])
	}
	if text == "" {
		text = collectText(m["text"])
	}
	return role, text
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// collectText gathers text from a string, or from "text" fields nested anywhere
// in arrays/objects (content-block arrays), joining them with spaces.
func collectText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	var b strings.Builder
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			if s, ok := t["text"].(string); ok {
				b.WriteString(s)
				b.WriteByte(' ')
			}
			if c, ok := t["content"]; ok {
				walk(c)
			}
		}
	}
	walk(v)
	return strings.TrimSpace(b.String())
}
