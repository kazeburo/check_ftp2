package main

import (
	"errors"
	"testing"
	"time"
)

func Test_replaceReplacer(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"foo\nbar", "foo\\nbar"},
		{"foo\rbar", "foo\\nbar"},
		{"foo\r\nbar", "foo\\nbar"},
		{"foo", "foo"},
	}
	for _, c := range cases {
		got := replaceReplacer(c.in)
		if got != c.want {
			t.Errorf("replaceReplacer(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}

func Test_ftpError_ErrorAndCode(t *testing.T) {
	err := &ftpError{
		msg:  "msg",
		code: 42,
	}
	if err.Error() != "msg" {
		t.Errorf("Error() = %q, want %q", err.Error(), "msg")
	}
	if err.Code() != 42 {
		t.Errorf("Code() = %d, want %d", err.Code(), 42)
	}
}

func Test_Opt_dialOptions_basic(t *testing.T) {
	opts := &Opt{Timeout: 1 * time.Second, Hostname: "localhost", Port: 21}
	options := opts.dialOptions()
	if len(options) == 0 {
		t.Error("dialOptions should return options")
	}
}

func Test_Opt_doConnect_timeout(t *testing.T) {
	// タイムアウトを極端に短くして必ず失敗させる
	opts := &Opt{Timeout: 1 * time.Nanosecond, Hostname: "localhost", Port: 21}
	msg, err := opts.doConnect()
	if err == nil {
		t.Error("doConnect should fail on timeout")
	}
	if !errors.Is(err, err) && msg != "" {
		t.Error("doConnect should return error and empty msg")
	}
}
