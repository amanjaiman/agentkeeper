package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["model"] != "llama3.1" || req["stream"] != false {
			t.Errorf("unexpected request body: %v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"response": "next: write the tests", "done": true})
	}))
	defer srv.Close()

	c := New(srv.URL)
	out, err := c.Generate(context.Background(), "llama3.1", "meta")
	if err != nil {
		t.Fatal(err)
	}
	if out != "next: write the tests" {
		t.Fatalf("response = %q", out)
	}
}

func TestGenerateHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := New(srv.URL).Generate(context.Background(), "missing", "x")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected an HTTP 404 error, got %v", err)
	}
}

func TestGenerateBodyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "out of memory"})
	}))
	defer srv.Close()

	_, err := New(srv.URL).Generate(context.Background(), "m", "x")
	if err == nil || !strings.Contains(err.Error(), "out of memory") {
		t.Fatalf("expected the model error surfaced, got %v", err)
	}
}

func TestNewNormalizesHost(t *testing.T) {
	if got := New("127.0.0.1:11434").BaseURL; got != "http://127.0.0.1:11434" {
		t.Fatalf("BaseURL = %q, want scheme prepended", got)
	}
	if got := New("http://h:1/").BaseURL; got != "http://h:1" {
		t.Fatalf("BaseURL = %q, want trailing slash trimmed", got)
	}
}
