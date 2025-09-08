package logger

import "testing"

func TestJSONEnabled(t *testing.T) {
	l := New(false)
	if l.JSONEnabled() {
		t.Fatal("expected false")
	}
	l = New(true)
	if !l.JSONEnabled() {
		t.Fatal("expected true")
	}
}
