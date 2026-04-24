package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeCustomGroups(t *testing.T) {
	got, err := NormalizeCustomGroups([]string{
		" unused ",
		"参考",
		"unused",
		"alpha.beta",
		"team_tools",
		"team-tools",
		"",
	})
	if err != nil {
		t.Fatalf("NormalizeCustomGroups returned error: %v", err)
	}
	want := []string{"alpha.beta", "team-tools", "team_tools", "unused", "参考"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestNormalizeCustomGroupsRejectsInvalidNames(t *testing.T) {
	tests := [][]string{
		{"has/slash"},
		{"has\\slash"},
		{"has\nnewline"},
		{string(make([]byte, 65))},
	}
	for _, input := range tests {
		if _, err := NormalizeCustomGroups(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestNormalizeCustomGroupsRejectsTooManyGroups(t *testing.T) {
	input := make([]string, MaxCustomGroupsPerSkill+1)
	for i := range input {
		input[i] = string(rune('a' + i))
	}
	if _, err := NormalizeCustomGroups(input); err == nil {
		t.Fatal("expected too many groups error")
	}
}

func TestMetadataEntryCustomGroupsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewMetadataStore()
	store.Set("local-skill", &MetadataEntry{CustomGroups: []string{"unused", "reference"}})
	if err := store.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadMetadata(dir)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	got := loaded.Get("local-skill")
	if got == nil {
		t.Fatal("expected metadata entry")
	}
	if len(got.CustomGroups) != 2 || got.CustomGroups[0] != "unused" || got.CustomGroups[1] != "reference" {
		t.Fatalf("custom groups = %v", got.CustomGroups)
	}
}

func TestMetadataEntryHasMetadataBeyondCustomGroups(t *testing.T) {
	if (&MetadataEntry{CustomGroups: []string{"unused"}}).HasMetadataBeyondCustomGroups() {
		t.Fatal("custom-groups-only entry should not have other metadata")
	}
	if !(&MetadataEntry{Source: "github.com/acme/skills", CustomGroups: []string{"unused"}}).HasMetadataBeyondCustomGroups() {
		t.Fatal("source-backed entry should have other metadata")
	}
	if !(&MetadataEntry{FileHashes: map[string]string{"SKILL.md": "sha256:abc"}}).HasMetadataBeyondCustomGroups() {
		t.Fatal("file hash metadata should count as other metadata")
	}
}

func TestLoadMetadataOldEntriesWithoutCustomGroups(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"version":1,"entries":{"alpha":{"source":"github.com/acme/alpha"}}}`)
	if err := os.WriteFile(filepath.Join(dir, MetadataFileName), data, 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	store, err := LoadMetadata(dir)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	entry := store.Get("alpha")
	if entry == nil || entry.Source != "github.com/acme/alpha" {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.CustomGroups != nil {
		t.Fatalf("expected nil custom groups for old entry, got %v", entry.CustomGroups)
	}

	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}
	if string(raw) != `{"source":"github.com/acme/alpha"}` {
		t.Fatalf("unexpected old-entry JSON shape: %s", raw)
	}
}
