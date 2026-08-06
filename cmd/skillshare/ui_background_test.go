package main

import (
	"reflect"
	"testing"
)

func TestSplitUIModeRecognizesBackgroundSubcommands(t *testing.T) {
	mode, rest := splitUIMode([]string{"start", "--port", "8080"})
	if mode != uiStartMode {
		t.Fatalf("mode = %q, want %q", mode, uiStartMode)
	}
	if !reflect.DeepEqual(rest, []string{"--port", "8080"}) {
		t.Fatalf("rest = %#v", rest)
	}

	mode, rest = splitUIMode([]string{"--port", "8080"})
	if mode != "" {
		t.Fatalf("mode = %q, want empty", mode)
	}
	if !reflect.DeepEqual(rest, []string{"--port", "8080"}) {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestUIChildArgsKeepsExistingUIAsForegroundServer(t *testing.T) {
	opts := uiBackgroundOptions{
		mode:     modeProject,
		host:     "0.0.0.0",
		port:     "8080",
		basePath: "/studio",
		noOpen:   true,
	}

	got := uiChildArgs(opts)
	want := []string{"ui", "--no-open", "--host", "0.0.0.0", "--port", "8080", "--base-path", "/studio", "--project"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uiChildArgs() = %#v, want %#v", got, want)
	}
}

func TestUIRestartArgsUseHiddenRestartCommand(t *testing.T) {
	opts := uiBackgroundOptions{mode: modeGlobal, host: "127.0.0.1", port: "19420", basePath: "/app", noOpen: true, clearCache: true}
	got := uiRestartArgs(opts)
	want := []string{"__ui-restart", "--no-open", "--host", "127.0.0.1", "--port", "19420", "--base-path", "/app", "--clear-cache", "--global"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uiRestartArgs() = %#v, want %#v", got, want)
	}
}
