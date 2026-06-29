package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultAdaptersCompile(t *testing.T) {
	cfg := Default()
	for _, name := range []string{"claude", "codex"} {
		if _, err := cfg.Adapter(name); err != nil {
			t.Errorf("default adapter %q does not compile: %v", name, err)
		}
	}
	// Codex must carry the relative-duration pattern added in M3.
	codex := cfg.Agents["codex"]
	var hasDur bool
	for _, p := range codex.LimitPatterns {
		if contains(p, "(?P<dur>") {
			hasDur = true
		}
	}
	if !hasDur {
		t.Errorf("codex adapter missing a (?P<dur>) relative-duration pattern: %v", codex.LimitPatterns)
	}
	// Both default adapters define a yolo flag for the explicit --yolo opt-in.
	for _, name := range []string{"claude", "codex"} {
		if cfg.Agents[name].YoloFlag == "" {
			t.Errorf("adapter %q has no yolo_flag configured", name)
		}
		ad, _ := cfg.Adapter(name)
		if ad.YoloFlag == "" {
			t.Errorf("compiled adapter %q dropped the yolo flag", name)
		}
	}
}

func TestDefaultClaudeAutoSelectsRateLimitMenu(t *testing.T) {
	ad, err := Default().Adapter("claude")
	if err != nil {
		t.Fatalf("claude adapter: %v", err)
	}
	if len(ad.AutoResponses) == 0 {
		t.Fatal("default claude adapter ships no auto_responses; the rate-limit menu won't be auto-answered")
	}
	ar := ad.AutoResponses[0]
	if ar.Keys != "1\r" {
		t.Errorf("rate-limit auto-response keys = %q, want %q", ar.Keys, "1\\r")
	}
	if !ar.Pattern.MatchString("1. Stop and wait for limit to reset") {
		t.Errorf("auto-response pattern %q does not match the stop-and-wait menu", ar.Pattern)
	}
}

func TestDefaultClaudePromptPattern(t *testing.T) {
	ad, err := Default().Adapter("claude")
	if err != nil {
		t.Fatalf("claude adapter: %v", err)
	}
	if ad.PromptPattern == nil {
		t.Fatal("default claude adapter has no prompt_pattern")
	}
	for _, name := range []string{
		"rate-limit-menu.txt",
		"permissions-menu.txt",
		"workspace-trust-prompt.txt",
	} {
		body := readClaudeTestdata(t, name)
		if !ad.PromptPattern.MatchString(body) {
			t.Fatalf("prompt_pattern did not match captured %s", name)
		}
	}
	negatives := []string{
		"Claude Code v2.1.195\nWelcome back\n* high - /effort\n",
		"Frosting...\nesc to interrupt\n",
		"> 1. write tests\n2. run them\n",
		"Here is a normal numbered list:\n1. first\n2. second\nNo prompt here.\n",
	}
	for _, body := range negatives {
		if ad.PromptPattern.MatchString(body) {
			t.Fatalf("prompt_pattern false-positive on %q", body)
		}
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err != nil {
		t.Fatalf("missing config should not error: %v", err)
	}
	if cfg.PollInterval.D() != 3*time.Second {
		t.Fatalf("poll_interval = %v, want default 3s", cfg.PollInterval.D())
	}
}

func TestOverlayReplacesAgentKeepsTimingsAndOtherAgents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `
poll_interval = "7s"

[agents.codex]
launch_cmd = "codex"
limit_patterns = ["(?i)custom (?P<time>.+)"]
inject_style = "text-enter"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PollInterval.D() != 7*time.Second {
		t.Errorf("poll_interval = %v, want overridden 7s", cfg.PollInterval.D())
	}
	if cfg.ResetBuffer.D() != 60*time.Second {
		t.Errorf("reset_buffer = %v, want default 60s preserved", cfg.ResetBuffer.D())
	}
	// User-supplied codex entry fully replaces the default one.
	if pats := cfg.Agents["codex"].LimitPatterns; len(pats) != 1 || pats[0] != "(?i)custom (?P<time>.+)" {
		t.Errorf("codex patterns = %v, want the single user pattern", pats)
	}
	// The default claude adapter is untouched.
	if _, err := cfg.Adapter("claude"); err != nil {
		t.Errorf("claude adapter should survive overlay: %v", err)
	}
}

func TestLoadAutoResponses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `
[agents.claude]
launch_cmd = "claude"
limit_patterns = ["(?i)limit reached (?P<time>.+)"]

[[agents.claude.auto_responses]]
pattern = "(?i)stop and wait for the limit to reset"
keys = "1\r"
once = true
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := cfg.Adapter("claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(ad.AutoResponses) != 1 {
		t.Fatalf("auto responses = %d, want 1", len(ad.AutoResponses))
	}
	if ad.AutoResponses[0].Keys != "1\r" || !ad.AutoResponses[0].Once {
		t.Fatalf("auto response = %+v", ad.AutoResponses[0])
	}
	if !ad.AutoResponses[0].Pattern.MatchString("Stop and wait for the limit to reset") {
		t.Fatal("compiled auto-response pattern did not match")
	}
}

func TestLoadPromptPattern(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `
[agents.fake]
launch_cmd = "fake"
limit_patterns = ["(?i)limit (?P<time>.+)"]
prompt_pattern = "(?i)choose one"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := cfg.Adapter("fake")
	if err != nil {
		t.Fatal(err)
	}
	if ad.PromptPattern == nil || !ad.PromptPattern.MatchString("Choose one") {
		t.Fatalf("compiled prompt pattern missing or does not match: %+v", ad.PromptPattern)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func readClaudeTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "parser", "testdata", "claude", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
