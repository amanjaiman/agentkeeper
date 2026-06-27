package ptybackend

import "regexp"

// ringSize bounds how much recent pty/conpty output is retained for Capture.
const ringSize = 64 * 1024

// ansi matches OSC sequences, CSI sequences, two-char escapes, CR and NUL — the
// control noise a real TUI emits, stripped so limit-string matching sees text.
var ansi = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)|\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b[@-Z\\-_]|[\r\x00]`)

func stripANSI(s string) string { return ansi.ReplaceAllString(s, "") }
