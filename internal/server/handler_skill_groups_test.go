package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"skillshare/internal/install"
)

func TestHandleListSkills_IncludesCustomGroups(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")
	store := install.NewMetadataStore()
	store.Set("alpha", &install.MetadataEntry{CustomGroups: []string{"reference", "unused"}})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	req := httptest.NewRequest(http.MethodGet, "/api/resources", nil)
	rr := httptest.NewRecorder()
	s.handleListSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Resources []struct {
			FlatName string   `json:"flatName"`
			Groups   []string `json:"groups"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resp.Resources))
	}
	if got := resp.Resources[0].Groups; len(got) != 2 || got[0] != "reference" || got[1] != "unused" {
		t.Fatalf("groups = %v", resp.Resources[0].Groups)
	}
}

func TestHandleSetSkillGroups_CreatesLightweightEntry(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")
	before, err := os.ReadFile(filepath.Join(src, "alpha", "SKILL.md"))
	if err != nil {
		t.Fatalf("read skill before: %v", err)
	}

	body := `{"groups":[" unused ","reference","unused"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	store, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	entry := store.Get("alpha")
	if entry == nil {
		t.Fatal("expected metadata entry")
	}
	if got := entry.CustomGroups; len(got) != 2 || got[0] != "reference" || got[1] != "unused" {
		t.Fatalf("groups = %v", got)
	}
	after, err := os.ReadFile(filepath.Join(src, "alpha", "SKILL.md"))
	if err != nil {
		t.Fatalf("read skill after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("SKILL.md changed, but custom groups must stay in metadata")
	}
}

func TestHandleSetSkillGroups_UsesRelPathForTrackedRepoChild(t *testing.T) {
	s, src := newTestServer(t)
	addSkillNested(t, src, "_team-skills/frontend/ui")

	body := `{"groups":["team","unused"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/_team-skills__frontend__ui/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "_team-skills__frontend__ui")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	store, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	if got := store.Get("_team-skills/frontend/ui"); got == nil || len(got.CustomGroups) != 2 {
		t.Fatalf("expected relPath metadata entry, got %+v", got)
	}
	if got := store.Get("ui"); got != nil {
		t.Fatalf("did not expect basename metadata entry, got %+v", got)
	}
}

func TestHandleSetSkillGroups_DoesNotMigrateTopLevelBasenameMetadata(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "ui")
	addSkillNested(t, src, "_team-skills/frontend/ui")
	store := install.NewMetadataStore()
	store.Set("ui", &install.MetadataEntry{Source: "github.com/acme/ui"})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	body := `{"groups":["team"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/_team-skills__frontend__ui/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "_team-skills__frontend__ui")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	reloaded, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	topLevel := reloaded.Get("ui")
	if topLevel == nil || topLevel.Source != "github.com/acme/ui" {
		t.Fatalf("expected top-level ui metadata to remain, got %+v", topLevel)
	}
	if len(topLevel.CustomGroups) != 0 {
		t.Fatalf("expected top-level groups unchanged, got %v", topLevel.CustomGroups)
	}
	nested := reloaded.Get("_team-skills/frontend/ui")
	if nested == nil {
		t.Fatal("expected nested relPath metadata entry")
	}
	if got := nested.CustomGroups; len(got) != 1 || got[0] != "team" {
		t.Fatalf("nested groups = %v", got)
	}
}

func TestHandleSetSkillGroups_SerializesConcurrentMetadataWrites(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")
	addSkill(t, src, "beta")

	for i := 0; i < 25; i++ {
		store := install.NewMetadataStore()
		if err := store.Save(src); err != nil {
			t.Fatalf("save metadata: %v", err)
		}
		s.skillsStore = store

		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make(chan string, 2)
		for _, tc := range []struct {
			name  string
			group string
		}{
			{name: "alpha", group: "one"},
			{name: "beta", group: "two"},
		} {
			tc := tc
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				body := `{"groups":["` + tc.group + `"]}`
				req := httptest.NewRequest(http.MethodPatch, "/api/resources/"+tc.name+"/groups", bytes.NewBufferString(body))
				req.SetPathValue("name", tc.name)
				rr := httptest.NewRecorder()
				s.handleSetSkillGroups(rr, req)
				if rr.Code != http.StatusOK {
					errs <- tc.name + ": " + rr.Body.String()
				}
			}()
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Fatal(err)
		}

		reloaded, err := install.LoadMetadata(src)
		if err != nil {
			t.Fatalf("load metadata: %v", err)
		}
		alpha := reloaded.Get("alpha")
		beta := reloaded.Get("beta")
		if alpha == nil || len(alpha.CustomGroups) != 1 || alpha.CustomGroups[0] != "one" ||
			beta == nil || len(beta.CustomGroups) != 1 || beta.CustomGroups[0] != "two" {
			t.Fatalf("iteration %d: expected both groups, alpha=%+v beta=%+v", i, alpha, beta)
		}
	}
}

func TestHandleSetSkillGroups_ClearsAndDeletesLightweightEntry(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")
	store := install.NewMetadataStore()
	store.Set("alpha", &install.MetadataEntry{CustomGroups: []string{"unused"}})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	body := `{"groups":[]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	reloaded, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	if reloaded.Get("alpha") != nil {
		t.Fatalf("expected custom-groups-only entry to be removed, got %+v", reloaded.Get("alpha"))
	}
}

func TestHandleSetSkillGroups_ClearsButKeepsSourceMetadata(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")
	store := install.NewMetadataStore()
	store.Set("alpha", &install.MetadataEntry{Source: "github.com/acme/alpha", CustomGroups: []string{"unused"}})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	body := `{"groups":[]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	reloaded, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	entry := reloaded.Get("alpha")
	if entry == nil || entry.Source != "github.com/acme/alpha" {
		t.Fatalf("expected source metadata to remain, got %+v", entry)
	}
	if len(entry.CustomGroups) != 0 {
		t.Fatalf("expected groups cleared, got %v", entry.CustomGroups)
	}
}

func TestHandleSetSkillGroups_RejectsInvalidNames(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")

	body := `{"groups":["bad/name"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "may contain only") {
		t.Fatalf("expected validation message, got %s", rr.Body.String())
	}
}

func TestHandleSetSkillGroups_RejectsInvalidKind(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")

	body := `{"groups":["unused"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups?kind=unknown", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid kind") {
		t.Fatalf("expected invalid kind message, got %s", rr.Body.String())
	}
}

func TestHandleSetSkillGroups_RejectsAgentKind(t *testing.T) {
	s, _ := newTestServer(t)
	body := `{"groups":["unused"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/reviewer.md/groups?kind=agent", bytes.NewBufferString(body))
	req.SetPathValue("name", "reviewer.md")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "custom groups only support skills") {
		t.Fatalf("expected agent rejection message, got %s", rr.Body.String())
	}
}

func TestHandleSetSkillGroups_NotFound(t *testing.T) {
	s, _ := newTestServer(t)
	body := `{"groups":["unused"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/missing/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "missing")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
