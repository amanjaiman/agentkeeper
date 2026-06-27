//go:build windows

package ptybackend

import (
	"strings"
	"testing"
	"time"
)

func captureUntil(t *testing.T, c *Client, want string, d time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(d)
	var cap string
	for time.Now().Before(deadline) {
		cap, _ = c.Capture(200)
		if strings.Contains(cap, want) {
			return cap
		}
		time.Sleep(150 * time.Millisecond)
	}
	return cap
}

func TestConPTYCaptureAndExit(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Skipf("ConPTY unavailable: %v", err)
	}
	if err := c.Start(`cmd /c echo CONPTY_HELLO_MARKER & ping -n 2 127.0.0.1 > nul`); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if got := captureUntil(t, c, "CONPTY_HELLO_MARKER", 5*time.Second); !strings.Contains(got, "CONPTY_HELLO_MARKER") {
		c.mu.Lock()
		raw := string(c.ring)
		c.mu.Unlock()
		t.Logf("RAW RING (%d bytes): %q", len(raw), raw)
		t.Fatalf("capture missing expected text; got: %q", got)
	}

	// After the child exits, Ended() must become true so the supervisor can stop.
	var ended bool
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if ended, _ = c.Ended(); ended {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !ended {
		t.Fatal("Ended() never became true after the child exited")
	}
}

func TestConPTYInject(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Skipf("ConPTY unavailable: %v", err)
	}
	// /v:on enables delayed expansion so !x! is evaluated after set /p runs.
	if err := c.Start(`cmd /v:on /c set /p x= & echo GOT-!x!`); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	time.Sleep(800 * time.Millisecond) // let cmd reach the set /p prompt
	if err := c.Inject("PINGVALUE", "text-enter"); err != nil {
		t.Fatal(err)
	}
	if got := captureUntil(t, c, "GOT-PINGVALUE", 5*time.Second); !strings.Contains(got, "GOT-PINGVALUE") {
		t.Fatalf("injected input not echoed; capture: %q", got)
	}
}
