package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	versioncheck "skillshare/internal/version"
)

func TestUIRestartHelperArgsPreservesAddressBasePathAndMode(t *testing.T) {
	s := &Server{addr: "0.0.0.0:19420", basePath: "/studio", projectRoot: "/tmp/project"}

	got, err := s.uiRestartHelperArgs(true)
	if err != nil {
		t.Fatalf("uiRestartHelperArgs returned error: %v", err)
	}
	want := []string{"__ui-restart", "--no-open", "--host", "0.0.0.0", "--port", "19420", "--base-path", "/studio", "--clear-cache", "--project"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestRestartRequestAllowedRejectsCrossSite(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/restart", strings.NewReader(`{}`))
	req.RemoteAddr = "203.0.113.10:1234"
	req.Host = "127.0.0.1:19420"
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	if restartRequestAllowed(req, "0.0.0.0:19420") {
		t.Fatal("cross-site restart request should be rejected")
	}
}

func TestHandleUpgradeUsesInjectedRunner(t *testing.T) {
	old := runUIUpgrade
	runUIUpgrade = func() (uiUpgradeResult, error) {
		return uiUpgradeResult{Updated: true, Output: "upgraded"}, nil
	}
	defer func() { runUIUpgrade = old }()

	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/upgrade", nil)

	s.handleUpgrade(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"updated":true`) {
		t.Fatalf("response should include updated=true: %s", rec.Body.String())
	}
}

func TestDefaultRunUIUpgradeSimulatesDevMode(t *testing.T) {
	oldVersion := versioncheck.Version
	versioncheck.Version = "dev"
	defer func() { versioncheck.Version = oldVersion }()

	result, err := defaultRunUIUpgrade()
	if err != nil {
		t.Fatalf("defaultRunUIUpgrade returned error: %v", err)
	}
	if !result.Updated || !result.DevMode || result.LatestVersion == "" {
		t.Fatalf("dev upgrade result = %#v", result)
	}
}

func TestHandleUpgradeReturnsRunnerError(t *testing.T) {
	old := runUIUpgrade
	runUIUpgrade = func() (uiUpgradeResult, error) {
		return uiUpgradeResult{}, errors.New("boom")
	}
	defer func() { runUIUpgrade = old }()

	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/upgrade", nil)

	s.handleUpgrade(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("response should include error: %s", rec.Body.String())
	}
}
