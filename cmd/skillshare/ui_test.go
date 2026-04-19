package main

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/ui"
	versionpkg "skillshare/internal/version"
)

func withEnsureUIAvailableTestHooks(
	t *testing.T,
	isCached func(string) (string, bool),
	download func(string) error,
	embedded func() bool,
) {
	t.Helper()

	oldIsCached := uiDistIsCachedFn
	oldDownload := uiDistDownloadFn
	oldEmbedded := embeddedUIAvailableFn

	uiDistIsCachedFn = isCached
	uiDistDownloadFn = download
	embeddedUIAvailableFn = embedded

	t.Cleanup(func() {
		uiDistIsCachedFn = oldIsCached
		uiDistDownloadFn = oldDownload
		embeddedUIAvailableFn = oldEmbedded
	})
}

func TestEnsureUIAvailable_EmbeddedUISkipsCacheAndDownload(t *testing.T) {
	oldVersion := versionpkg.Version
	versionpkg.Version = "9.9.9-test"
	t.Cleanup(func() {
		versionpkg.Version = oldVersion
	})

	var cacheChecks int
	var downloadCalls int
	withEnsureUIAvailableTestHooks(
		t,
		func(string) (string, bool) {
			cacheChecks++
			return "", false
		},
		func(string) error {
			downloadCalls++
			return nil
		},
		func() bool { return true },
	)

	dir, err := ensureUIAvailable()
	if err != nil {
		t.Fatalf("ensureUIAvailable() error = %v", err)
	}
	if dir != "" {
		t.Fatalf("ensureUIAvailable() dir = %q, want empty dir for embedded UI", dir)
	}
	if cacheChecks != 0 {
		t.Fatalf("cache checks = %d, want 0 when embedded UI is available", cacheChecks)
	}
	if downloadCalls != 0 {
		t.Fatalf("download calls = %d, want 0 when embedded UI is available", downloadCalls)
	}
}

func TestEnsureUIAvailable_DownloadsWhenEmbeddedUIUnavailable(t *testing.T) {
	oldVersion := versionpkg.Version
	versionpkg.Version = "9.9.9-test"
	t.Cleanup(func() {
		versionpkg.Version = oldVersion
	})

	progressFile, err := os.Create(filepath.Join(t.TempDir(), "progress.log"))
	if err != nil {
		t.Fatalf("Create(progress log) error = %v", err)
	}
	defer progressFile.Close()

	oldProgressWriter := ui.ProgressWriter
	ui.SetProgressWriter(progressFile)
	ui.SuppressProgress()
	t.Cleanup(func() {
		ui.SetProgressWriter(oldProgressWriter)
		ui.RestoreProgress()
	})

	var cacheChecks int
	var downloadCalls int
	const cachedDir = "/tmp/skillshare-ui"
	withEnsureUIAvailableTestHooks(
		t,
		func(ver string) (string, bool) {
			cacheChecks++
			switch cacheChecks {
			case 1:
				return "", false
			case 2:
				return cachedDir, true
			default:
				t.Fatalf("unexpected cache check #%d for version %q", cacheChecks, ver)
				return "", false
			}
		},
		func(ver string) error {
			downloadCalls++
			if ver != "9.9.9-test" {
				t.Fatalf("download version = %q, want %q", ver, "9.9.9-test")
			}
			return nil
		},
		func() bool { return false },
	)

	dir, err := ensureUIAvailable()
	if err != nil {
		t.Fatalf("ensureUIAvailable() error = %v", err)
	}
	if dir != cachedDir {
		t.Fatalf("ensureUIAvailable() dir = %q, want %q", dir, cachedDir)
	}
	if cacheChecks != 2 {
		t.Fatalf("cache checks = %d, want 2", cacheChecks)
	}
	if downloadCalls != 1 {
		t.Fatalf("download calls = %d, want 1", downloadCalls)
	}
}
