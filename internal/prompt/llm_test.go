package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type fakeGen struct {
	out    string
	err    error
	called bool
}

func (f *fakeGen) Generate(_ context.Context, _, _ string) (string, error) {
	f.called = true
	return f.out, f.err
}

func writeTranscript(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	if err := os.WriteFile(path, []byte(`{"role":"user","content":"implement the parser"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "*.jsonl")
}

func newLLM(t *testing.T, gen Generator) LLM {
	return LLM{
		Model:          "test",
		Client:         gen,
		TranscriptGlob: writeTranscript(t),
		TailMessages:   10,
		MaxChars:       200,
		Denylist:       []string{"rm -rf", "--force"},
		Fallback:       "STATIC",
		gitContext:     func(string) string { return "" }, // no git in tests
	}
}

func TestLLMHappyPathSanitizes(t *testing.T) {
	l := newLLM(t, &fakeGen{out: "  \"Finish the parser in parser.go\"  \n"})
	got, err := l.Build(Context{Cwd: "/x"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Finish the parser in parser.go" {
		t.Fatalf("got %q, want sanitized single-line instruction", got)
	}
}

func TestLLMDenylistFallsBack(t *testing.T) {
	gen := &fakeGen{out: "run rm -rf build to clean then rebuild"}
	l := newLLM(t, gen)
	got, _ := l.Build(Context{Cwd: "/x"})
	if got != "STATIC" {
		t.Fatalf("got %q, want fallback to STATIC for denylisted output", got)
	}
}

func TestLLMTooLongFallsBack(t *testing.T) {
	l := newLLM(t, &fakeGen{out: "x to the power of many words repeated"})
	l.MaxChars = 5
	got, _ := l.Build(Context{Cwd: "/x"})
	if got != "STATIC" {
		t.Fatalf("got %q, want fallback for over-long output", got)
	}
}

func TestLLMServerErrorFallsBack(t *testing.T) {
	l := newLLM(t, &fakeGen{err: fmt.Errorf("connection refused")})
	got, _ := l.Build(Context{Cwd: "/x"})
	if got != "STATIC" {
		t.Fatalf("got %q, want fallback when the model errors", got)
	}
}

func TestLLMNoContextDoesNotCallModel(t *testing.T) {
	gen := &fakeGen{out: "should not be used"}
	l := LLM{
		Model:          "test",
		Client:         gen,
		TranscriptGlob: filepath.Join(t.TempDir(), "*.jsonl"), // matches nothing
		Fallback:       "STATIC",
		gitContext:     func(string) string { return "" },
	}
	got, _ := l.Build(Context{Cwd: "/x"})
	if got != "STATIC" {
		t.Fatalf("got %q, want fallback with no context", got)
	}
	if gen.called {
		t.Fatal("model should not be called when there is no context to summarize")
	}
}

func TestLLMEmptyOutputFallsBack(t *testing.T) {
	l := newLLM(t, &fakeGen{out: "   \n  "})
	got, _ := l.Build(Context{Cwd: "/x"})
	if got != "STATIC" {
		t.Fatalf("got %q, want fallback for empty output", got)
	}
}
