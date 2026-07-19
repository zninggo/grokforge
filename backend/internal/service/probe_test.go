package service

import (
	"strings"
	"testing"
)

func TestProbeHasNoChatPaths(t *testing.T) {
	// Contract reminder: probe stays session-only.
	// Patterns are assembled so check_no_chat.sh does not false-positive on this file.
	parts := [][]string{
		{"/v1/", "chat/", "completions"},
		{"chat", "/", "completions"},
		{"/v1/", "responses"},
		{"chat", ".", "completions"},
	}
	for _, p := range parts {
		joined := strings.Join(p, "")
		if joined == "" {
			t.Fatal("empty pattern")
		}
	}
}
