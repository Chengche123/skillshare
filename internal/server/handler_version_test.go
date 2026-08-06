package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"skillshare/internal/config"
	versioncheck "skillshare/internal/version"
)

func TestHandleVersionCheckDevModeExposesSimulatedUpdate(t *testing.T) {
	oldVersion := versioncheck.Version
	versioncheck.Version = "dev"
	defer func() { versioncheck.Version = oldVersion }()

	s := &Server{cfg: &config.Config{Source: t.TempDir()}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)

	s.handleVersionCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["cliDevMode"] != true || body["cliUpdateAvailable"] != true || body["cliLatest"] == "" {
		t.Fatalf("dev version response = %#v", body)
	}
}
