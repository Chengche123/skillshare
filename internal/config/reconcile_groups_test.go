package config

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/install"
)

func writeSkillDir(t *testing.T, sourcePath, relPath string) {
	t.Helper()
	dir := filepath.Join(sourcePath, filepath.FromSlash(relPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+filepath.Base(filepath.FromSlash(relPath))+"\n---\n# skill\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func TestReconcileGlobalSkills_PreservesCustomGroupOnlyLocalSkill(t *testing.T) {
	sourcePath := t.TempDir()
	writeSkillDir(t, sourcePath, "local-skill")

	cfg := &Config{Source: sourcePath}
	store := install.NewMetadataStore()
	store.Set("local-skill", &install.MetadataEntry{CustomGroups: []string{"unused"}})

	if err := ReconcileGlobalSkills(cfg, store); err != nil {
		t.Fatalf("ReconcileGlobalSkills failed: %v", err)
	}
	entry := store.Get("local-skill")
	if entry == nil {
		t.Fatal("expected custom group metadata to survive reconcile")
	}
	if len(entry.CustomGroups) != 1 || entry.CustomGroups[0] != "unused" {
		t.Fatalf("groups = %v", entry.CustomGroups)
	}
}

func TestReconcileGlobalSkills_PreservesUnambiguousLegacyBasenameCustomGroups(t *testing.T) {
	sourcePath := t.TempDir()
	writeSkillDir(t, sourcePath, "frontend/ui")

	cfg := &Config{Source: sourcePath}
	store := install.NewMetadataStore()
	store.Set("ui", &install.MetadataEntry{CustomGroups: []string{"legacy"}})

	if err := ReconcileGlobalSkills(cfg, store); err != nil {
		t.Fatalf("ReconcileGlobalSkills failed: %v", err)
	}
	if old := store.Get("ui"); old != nil {
		t.Fatalf("expected legacy basename metadata to be migrated, got %+v", old)
	}
	entry := store.Get("frontend/ui")
	if entry == nil {
		t.Fatal("expected custom group metadata to survive under full path")
	}
	if got := entry.CustomGroups; len(got) != 1 || got[0] != "legacy" {
		t.Fatalf("groups = %v", got)
	}
}

func TestReconcileGlobalSkills_DoesNotLeakTopLevelGroupsToNestedBasename(t *testing.T) {
	sourcePath := t.TempDir()
	writeSkillDir(t, sourcePath, "ui")
	writeSkillDir(t, sourcePath, "frontend/ui")

	cfg := &Config{Source: sourcePath}
	store := install.NewMetadataStore()
	store.Set("ui", &install.MetadataEntry{CustomGroups: []string{"top-level"}})

	if err := ReconcileGlobalSkills(cfg, store); err != nil {
		t.Fatalf("ReconcileGlobalSkills failed: %v", err)
	}
	top := store.Get("ui")
	if top == nil || len(top.CustomGroups) != 1 || top.CustomGroups[0] != "top-level" {
		t.Fatalf("top-level metadata changed: %+v", top)
	}
	if nested := store.Get("frontend/ui"); nested != nil && len(nested.CustomGroups) != 0 {
		t.Fatalf("nested skill inherited top-level groups: %+v", nested)
	}
}

func TestReconcileGlobalSkills_PrunesCustomGroupOnlyMissingSkill(t *testing.T) {
	sourcePath := t.TempDir()
	cfg := &Config{Source: sourcePath}
	store := install.NewMetadataStore()
	store.Set("missing-skill", &install.MetadataEntry{CustomGroups: []string{"unused"}})

	if err := ReconcileGlobalSkills(cfg, store); err != nil {
		t.Fatalf("ReconcileGlobalSkills failed: %v", err)
	}
	if got := store.Get("missing-skill"); got != nil {
		t.Fatalf("expected missing custom group entry to be pruned, got %+v", got)
	}
}
