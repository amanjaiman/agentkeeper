package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTailExtractsRolesAndText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	write(t, path, strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"do task X"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"done part 1"},{"type":"tool_use","name":"Edit"}]}}`,
		`{"role":"user","content":"flat content here"}`,
		``, // blank line ignored
	}, "\n"))

	out, err := Tail(filepath.Join(dir, "*.jsonl"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "done part 1") || !strings.Contains(out, "flat content here") {
		t.Fatalf("tail missing recent messages:\n%s", out)
	}
	if strings.Contains(out, "do task X") {
		t.Fatalf("tail should be limited to last 2 messages:\n%s", out)
	}
	if !strings.Contains(out, "assistant:") || !strings.Contains(out, "user:") {
		t.Fatalf("tail missing roles:\n%s", out)
	}
}

func TestTailPicksNewestFile(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.jsonl")
	cur := filepath.Join(dir, "cur.jsonl")
	write(t, old, `{"role":"user","content":"OLD"}`+"\n")
	write(t, cur, `{"role":"user","content":"NEW"}`+"\n")
	// Make old genuinely older.
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}

	out, err := Tail(filepath.Join(dir, "*.jsonl"), 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "NEW") || strings.Contains(out, "OLD") {
		t.Fatalf("expected newest file content only, got: %s", out)
	}
}

func TestTailDoubleStarGlob(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "2026", "06", "deep.jsonl")
	write(t, nested, `{"role":"assistant","content":"buried message"}`+"\n")

	out, err := Tail(filepath.Join(dir, "**", "*.jsonl"), 5)
	if err != nil {
		t.Fatalf("double-star glob failed: %v", err)
	}
	if !strings.Contains(out, "buried message") {
		t.Fatalf("expected nested file via **, got: %s", out)
	}
}

func TestTailNoMatch(t *testing.T) {
	if _, err := Tail(filepath.Join(t.TempDir(), "*.jsonl"), 5); err == nil {
		t.Fatal("expected an error when no transcript matches")
	}
}
