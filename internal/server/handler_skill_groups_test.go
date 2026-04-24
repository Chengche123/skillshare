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

	"skillshare/internal/config"
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

func TestHandleGetSkill_IncludesCustomGroups(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")
	store := install.NewMetadataStore()
	store.Set("alpha", &install.MetadataEntry{CustomGroups: []string{"reference"}})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	req := httptest.NewRequest(http.MethodGet, "/api/resources/alpha", nil)
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleGetSkill(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Resource struct {
			Groups []string `json:"groups"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := resp.Resource.Groups; len(got) != 1 || got[0] != "reference" {
		t.Fatalf("groups = %v", got)
	}
}

func TestHandleListSkills_DoesNotLeakTopLevelGroupsToNestedBasename(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "ui")
	addSkillNested(t, src, "_team-skills/frontend/ui")
	store := install.NewMetadataStore()
	store.Set("ui", &install.MetadataEntry{CustomGroups: []string{"top-level"}})
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
	groupsByName := map[string][]string{}
	for _, item := range resp.Resources {
		groupsByName[item.FlatName] = item.Groups
	}
	if got := groupsByName["ui"]; len(got) != 1 || got[0] != "top-level" {
		t.Fatalf("top-level groups = %v", got)
	}
	if got := groupsByName["_team-skills__frontend__ui"]; len(got) != 0 {
		t.Fatalf("nested skill inherited top-level groups: %v", got)
	}
}

func TestHandleGetSkill_DoesNotLeakTopLevelGroupsToNestedBasename(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "ui")
	addSkillNested(t, src, "_team-skills/frontend/ui")
	store := install.NewMetadataStore()
	store.Set("ui", &install.MetadataEntry{CustomGroups: []string{"top-level"}})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	req := httptest.NewRequest(http.MethodGet, "/api/resources/_team-skills__frontend__ui", nil)
	req.SetPathValue("name", "_team-skills__frontend__ui")
	rr := httptest.NewRecorder()
	s.handleGetSkill(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Resource struct {
			Groups []string `json:"groups"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Resource.Groups) != 0 {
		t.Fatalf("nested skill inherited top-level groups: %v", resp.Resource.Groups)
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

func TestHandleSetSkillGroups_ProjectModeUsesProjectSkillsSource(t *testing.T) {
	projectRoot := t.TempDir()
	projectSkills := filepath.Join(projectRoot, ".skillshare", "skills")
	globalSkills := filepath.Join(t.TempDir(), "global-skills")
	if err := os.MkdirAll(projectSkills, 0o755); err != nil {
		t.Fatalf("mkdir project skills: %v", err)
	}
	if err := os.MkdirAll(globalSkills, 0o755); err != nil {
		t.Fatalf("mkdir global skills: %v", err)
	}
	addSkill(t, projectSkills, "alpha")
	addSkill(t, globalSkills, "alpha")
	if err := install.NewMetadataStore().Save(projectSkills); err != nil {
		t.Fatalf("save project metadata: %v", err)
	}
	if err := install.NewMetadataStore().Save(globalSkills); err != nil {
		t.Fatalf("save global metadata: %v", err)
	}

	s := NewProject(
		&config.Config{Source: globalSkills, Targets: map[string]config.TargetConfig{}},
		&config.ProjectConfig{},
		projectRoot,
		"127.0.0.1:0",
		"",
		"",
	)

	body := `{"groups":["project-only"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	projectStore, err := install.LoadMetadata(projectSkills)
	if err != nil {
		t.Fatalf("load project metadata: %v", err)
	}
	projectEntry := projectStore.Get("alpha")
	if projectEntry == nil || len(projectEntry.CustomGroups) != 1 || projectEntry.CustomGroups[0] != "project-only" {
		t.Fatalf("expected project metadata to be updated, got %+v", projectEntry)
	}
	globalStore, err := install.LoadMetadata(globalSkills)
	if err != nil {
		t.Fatalf("load global metadata: %v", err)
	}
	if globalEntry := globalStore.Get("alpha"); globalEntry != nil && len(globalEntry.CustomGroups) != 0 {
		t.Fatalf("global metadata was modified: %+v", globalEntry)
	}
}

func newProjectModeSourceSelectionServer(t *testing.T) (*Server, string) {
	t.Helper()
	projectRoot := t.TempDir()
	projectSkills := filepath.Join(projectRoot, ".skillshare", "skills")
	globalSkills := filepath.Join(t.TempDir(), "global-skills")
	if err := os.MkdirAll(projectSkills, 0o755); err != nil {
		t.Fatalf("mkdir project skills: %v", err)
	}
	if err := os.MkdirAll(globalSkills, 0o755); err != nil {
		t.Fatalf("mkdir global skills: %v", err)
	}
	addSkill(t, projectSkills, "alpha")
	addSkill(t, globalSkills, "alpha")
	addSkill(t, globalSkills, "beta")

	projectStore := install.NewMetadataStore()
	projectStore.Set("alpha", &install.MetadataEntry{CustomGroups: []string{"project-only"}})
	if err := projectStore.Save(projectSkills); err != nil {
		t.Fatalf("save project metadata: %v", err)
	}
	globalStore := install.NewMetadataStore()
	globalStore.Set("alpha", &install.MetadataEntry{CustomGroups: []string{"global-only"}})
	globalStore.Set("beta", &install.MetadataEntry{CustomGroups: []string{"global-only"}})
	if err := globalStore.Save(globalSkills); err != nil {
		t.Fatalf("save global metadata: %v", err)
	}

	s := NewProject(
		&config.Config{Source: globalSkills, Targets: map[string]config.TargetConfig{}},
		&config.ProjectConfig{},
		projectRoot,
		"127.0.0.1:0",
		"",
		"",
	)
	return s, projectSkills
}

func TestHandleListSkills_ProjectModeUsesProjectSkillsSource(t *testing.T) {
	s, projectSkills := newProjectModeSourceSelectionServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/resources", nil)
	rr := httptest.NewRecorder()
	s.handleListSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Resources []struct {
			FlatName   string   `json:"flatName"`
			SourcePath string   `json:"sourcePath"`
			Groups     []string `json:"groups"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Resources) != 1 {
		t.Fatalf("expected only project skill, got %d resources: %+v", len(resp.Resources), resp.Resources)
	}
	item := resp.Resources[0]
	if item.FlatName != "alpha" {
		t.Fatalf("expected alpha, got %q", item.FlatName)
	}
	if filepath.Clean(item.SourcePath) != filepath.Join(projectSkills, "alpha") {
		t.Fatalf("sourcePath = %q, want project source", item.SourcePath)
	}
	if got := item.Groups; len(got) != 1 || got[0] != "project-only" {
		t.Fatalf("groups = %v", got)
	}
}

func TestHandleGetSkill_ProjectModeUsesProjectSkillsSource(t *testing.T) {
	s, projectSkills := newProjectModeSourceSelectionServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/resources/alpha", nil)
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleGetSkill(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Resource struct {
			SourcePath string   `json:"sourcePath"`
			Groups     []string `json:"groups"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if filepath.Clean(resp.Resource.SourcePath) != filepath.Join(projectSkills, "alpha") {
		t.Fatalf("sourcePath = %q, want project source", resp.Resource.SourcePath)
	}
	if got := resp.Resource.Groups; len(got) != 1 || got[0] != "project-only" {
		t.Fatalf("groups = %v", got)
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

func TestHandleSetSkillGroups_SafelyMigratesUnambiguousBasenameMetadata(t *testing.T) {
	s, src := newTestServer(t)
	addSkillNested(t, src, "_team-skills/frontend/ui")
	store := install.NewMetadataStore()
	store.Set("ui", &install.MetadataEntry{
		Source:  "github.com/acme/team-skills",
		Version: "abc123",
	})
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
	if old := reloaded.Get("ui"); old != nil {
		t.Fatalf("expected legacy basename metadata to be migrated, got %+v", old)
	}
	entry := reloaded.Get("_team-skills/frontend/ui")
	if entry == nil {
		t.Fatal("expected relPath metadata entry")
	}
	if entry.Source != "github.com/acme/team-skills" || entry.Version != "abc123" {
		t.Fatalf("expected source metadata to migrate, got %+v", entry)
	}
	if got := entry.CustomGroups; len(got) != 1 || got[0] != "team" {
		t.Fatalf("groups = %v", got)
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

func TestHandleSetSkillTargets_PreservesGroupsWithStaleStore(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")

	stale := install.NewMetadataStore()
	body := `{"groups":["unused"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	s.skillsStore = stale
	targetReq := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/targets", bytes.NewBufferString(`{"target":"claude"}`))
	targetReq.SetPathValue("name", "alpha")
	targetRR := httptest.NewRecorder()
	s.handleSetSkillTargets(targetRR, targetReq)
	if targetRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", targetRR.Code, targetRR.Body.String())
	}

	reloaded, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	entry := reloaded.Get("alpha")
	if entry == nil {
		t.Fatal("expected metadata entry")
	}
	if got := entry.CustomGroups; len(got) != 1 || got[0] != "unused" {
		t.Fatalf("groups = %v", got)
	}
}

func TestHandleBatchSetTargets_PreservesGroupsWithStaleStore(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")

	stale := install.NewMetadataStore()
	body := `{"groups":["unused"]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	s.skillsStore = stale
	batchReq := httptest.NewRequest(http.MethodPost, "/api/resources/batch/targets", bytes.NewBufferString(`{"folder":"*","target":"claude"}`))
	batchRR := httptest.NewRecorder()
	s.handleBatchSetTargets(batchRR, batchReq)
	if batchRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", batchRR.Code, batchRR.Body.String())
	}

	reloaded, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	entry := reloaded.Get("alpha")
	if entry == nil {
		t.Fatal("expected metadata entry")
	}
	if got := entry.CustomGroups; len(got) != 1 || got[0] != "unused" {
		t.Fatalf("groups = %v", got)
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

func TestHandleSetSkillGroups_ClearsAndDeletesLegacyGroupedLightweightEntry(t *testing.T) {
	s, src := newTestServer(t)
	addSkillNested(t, src, "frontend/ui")
	store := install.NewMetadataStore()
	store.Set("ui", &install.MetadataEntry{Group: "frontend", CustomGroups: []string{"unused"}})
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	body := `{"groups":[]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/resources/frontend__ui/groups", bytes.NewBufferString(body))
	req.SetPathValue("name", "frontend__ui")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	reloaded, err := install.LoadMetadata(src)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	if reloaded.Get("ui") != nil || reloaded.Get("frontend/ui") != nil {
		t.Fatalf("expected legacy grouped custom-groups-only entry to be removed, got ui=%+v rel=%+v", reloaded.Get("ui"), reloaded.Get("frontend/ui"))
	}
}

func TestHandleSetSkillGroups_ConcurrentWithList(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")
	store := install.NewMetadataStore()
	if err := store.Save(src); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	s.skillsStore = store

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan string, 100)
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			<-start
			body := `{"groups":["group-` + string(rune('a'+(i%10))) + `"]}`
			req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(body))
			req.SetPathValue("name", "alpha")
			rr := httptest.NewRecorder()
			s.handleSetSkillGroups(rr, req)
			if rr.Code != http.StatusOK {
				errs <- rr.Body.String()
			}
		}(i)
		go func() {
			defer wg.Done()
			<-start
			req := httptest.NewRequest(http.MethodGet, "/api/resources", nil)
			rr := httptest.NewRecorder()
			s.handleListSkills(rr, req)
			if rr.Code != http.StatusOK {
				errs <- rr.Body.String()
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
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

func TestHandleSetSkillGroups_RejectsMissingGroups(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "missing field", body: `{}`},
		{name: "null field", body: `{"groups":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, src := newTestServer(t)
			addSkill(t, src, "alpha")
			store := install.NewMetadataStore()
			store.Set("alpha", &install.MetadataEntry{CustomGroups: []string{"kept"}})
			if err := store.Save(src); err != nil {
				t.Fatalf("save metadata: %v", err)
			}
			s.skillsStore = store

			req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(tc.body))
			req.SetPathValue("name", "alpha")
			rr := httptest.NewRecorder()
			s.handleSetSkillGroups(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "groups is required") {
				t.Fatalf("expected groups required message, got %s", rr.Body.String())
			}
			reloaded, err := install.LoadMetadata(src)
			if err != nil {
				t.Fatalf("load metadata: %v", err)
			}
			entry := reloaded.Get("alpha")
			if entry == nil || len(entry.CustomGroups) != 1 || entry.CustomGroups[0] != "kept" {
				t.Fatalf("expected existing groups to remain, got %+v", entry)
			}
		})
	}
}

func TestHandleSetSkillGroups_RejectsNonArrayGroups(t *testing.T) {
	s, src := newTestServer(t)
	addSkill(t, src, "alpha")

	req := httptest.NewRequest(http.MethodPatch, "/api/resources/alpha/groups", bytes.NewBufferString(`{"groups":"unused"}`))
	req.SetPathValue("name", "alpha")
	rr := httptest.NewRecorder()
	s.handleSetSkillGroups(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid request body") {
		t.Fatalf("expected invalid request body message, got %s", rr.Body.String())
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
