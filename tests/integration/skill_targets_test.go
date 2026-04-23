//go:build !online

package integration

import (
	"path/filepath"
	"testing"

	"skillshare/internal/testutil"
)

func TestSkillTargets_DeclaredTargetsDoNotRestrictSync(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.CreateSkill("claude-skill", map[string]string{
		"SKILL.md": "---\nname: claude-skill\ntargets: [claude]\n---\n# Claude only",
	})
	sb.CreateSkill("universal-skill", map[string]string{
		"SKILL.md": "---\nname: universal-skill\n---\n# Universal",
	})

	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	result := sb.RunCLI("sync")
	result.AssertSuccess(t)

	// Declared targets are preserved as metadata only and should not restrict sync.
	if !sb.IsSymlink(filepath.Join(claudePath, "claude-skill")) {
		t.Error("claude-skill should be synced to claude target")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "claude-skill")) {
		t.Error("claude-skill should also be synced to cursor target")
	}

	// universal-skill (no targets field) should be in both
	if !sb.IsSymlink(filepath.Join(claudePath, "universal-skill")) {
		t.Error("universal-skill should be synced to claude target")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "universal-skill")) {
		t.Error("universal-skill should be synced to cursor target")
	}
}

func TestSkillTargets_CrossModeMatching(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	// Declared targets should not block project-mode sync either.
	sb.CreateSkill("cross-skill", map[string]string{
		"SKILL.md": "---\nname: cross-skill\ntargets: [claude]\n---\n# Cross",
	})

	projectRoot := sb.SetupProjectDir("claude")
	sb.CreateProjectSkill(projectRoot, "cross-skill", map[string]string{
		"SKILL.md": "---\nname: cross-skill\ntargets: [claude]\n---\n# Cross",
	})

	result := sb.RunCLIInDir(projectRoot, "sync", "-p")
	result.AssertSuccess(t)

	// claude target path
	targetPath := filepath.Join(projectRoot, ".claude", "skills")
	if !sb.IsSymlink(filepath.Join(targetPath, "cross-skill")) {
		t.Error("skill with declared targets should still sync in project mode")
	}
}

func TestSkillTargets_MultipleDeclaredTargetsDoNotRestrictSync(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.CreateSkill("multi-skill", map[string]string{
		"SKILL.md": "---\nname: multi-skill\ntargets: [claude, cursor]\n---\n# Multi",
	})
	sb.CreateSkill("single-skill", map[string]string{
		"SKILL.md": "---\nname: single-skill\ntargets: [cursor]\n---\n# Single",
	})

	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	result := sb.RunCLI("sync")
	result.AssertSuccess(t)

	// multi-skill should be in both
	if !sb.IsSymlink(filepath.Join(claudePath, "multi-skill")) {
		t.Error("multi-skill should be in claude")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "multi-skill")) {
		t.Error("multi-skill should be in cursor")
	}

	// single-skill should also be synced to all configured targets.
	if !sb.IsSymlink(filepath.Join(claudePath, "single-skill")) {
		t.Error("single-skill should also be in claude")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "single-skill")) {
		t.Error("single-skill should be in cursor")
	}
}

func TestSkillTargets_DoctorNoDriftWarning(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.CreateSkill("claude-only", map[string]string{
		"SKILL.md": "---\nname: claude-only\ntargets: [claude]\n---\n# Claude only",
	})
	sb.CreateSkill("universal", map[string]string{
		"SKILL.md": "---\nname: universal\n---\n# Universal",
	})

	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	sb.RunCLI("sync").AssertSuccess(t)

	// Doctor should NOT warn about drift when targets are metadata only.
	result := sb.RunCLI("doctor")
	result.AssertSuccess(t)
	result.AssertOutputNotContains(t, "not synced")
}

func TestSkillTargets_StatusNoDriftWarning(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.CreateSkill("claude-only", map[string]string{
		"SKILL.md": "---\nname: claude-only\ntargets: [claude]\n---\n# Claude only",
	})
	sb.CreateSkill("universal", map[string]string{
		"SKILL.md": "---\nname: universal\n---\n# Universal",
	})

	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	sb.RunCLI("sync").AssertSuccess(t)

	// Status should NOT warn about drift when targets are metadata only.
	result := sb.RunCLI("status")
	result.AssertSuccess(t)
	result.AssertOutputNotContains(t, "not synced")
}

func TestSkillTargets_DoesNotPruneWhenDeclaredTargetsChange(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	// First sync with universal skill
	sb.CreateSkill("my-skill", map[string]string{
		"SKILL.md": "---\nname: my-skill\n---\n# Universal",
	})
	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	sb.RunCLI("sync").AssertSuccess(t)
	if !sb.IsSymlink(filepath.Join(cursorPath, "my-skill")) {
		t.Fatal("my-skill should be in cursor after first sync")
	}

	// Update SKILL.md to add declared targets; sync behavior should stay unchanged.
	sb.CreateSkill("my-skill", map[string]string{
		"SKILL.md": "---\nname: my-skill\ntargets: [claude]\n---\n# Claude only now",
	})

	sb.RunCLI("sync").AssertSuccess(t)
	if !sb.IsSymlink(filepath.Join(claudePath, "my-skill")) {
		t.Error("my-skill should still be in claude")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "my-skill")) {
		t.Error("my-skill should remain in cursor after adding declared targets")
	}
}

// --- metadata.targets integration tests ---

func TestSkillTargets_MetadataTargetsDoNotRestrictSync(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	// Use new metadata.targets format
	sb.CreateSkill("meta-skill", map[string]string{
		"SKILL.md": "---\nname: meta-skill\nmetadata:\n  targets: [claude]\n---\n# Metadata targets",
	})
	sb.CreateSkill("universal", map[string]string{
		"SKILL.md": "---\nname: universal\n---\n# Universal",
	})

	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	result := sb.RunCLI("sync")
	result.AssertSuccess(t)

	// metadata.targets is preserved as metadata only and should not restrict sync.
	if !sb.IsSymlink(filepath.Join(claudePath, "meta-skill")) {
		t.Error("meta-skill should be synced to claude target")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "meta-skill")) {
		t.Error("meta-skill should also be synced to cursor target")
	}

	// universal should be in both
	if !sb.IsSymlink(filepath.Join(claudePath, "universal")) {
		t.Error("universal should be synced to claude")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "universal")) {
		t.Error("universal should be synced to cursor")
	}
}

func TestSkillTargets_MetadataAndLegacyTargetsDoNotRestrictSync(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	// One skill uses metadata.targets, another uses top-level targets
	sb.CreateSkill("new-format", map[string]string{
		"SKILL.md": "---\nname: new-format\nmetadata:\n  targets: [claude]\n---\n# New format",
	})
	sb.CreateSkill("old-format", map[string]string{
		"SKILL.md": "---\nname: old-format\ntargets: [cursor]\n---\n# Old format",
	})

	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	result := sb.RunCLI("sync")
	result.AssertSuccess(t)

	// Neither metadata.targets nor top-level targets should restrict sync.
	if !sb.IsSymlink(filepath.Join(claudePath, "new-format")) {
		t.Error("new-format should be in claude")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "new-format")) {
		t.Error("new-format should also be in cursor")
	}

	if !sb.IsSymlink(filepath.Join(claudePath, "old-format")) {
		t.Error("old-format should also be in claude")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "old-format")) {
		t.Error("old-format should be in cursor")
	}
}

func TestSkillTargets_MetadataAndTopLevelTargetsDoNotRestrictSync(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	// Both top-level and metadata.targets may be present, but neither restricts sync.
	sb.CreateSkill("override-skill", map[string]string{
		"SKILL.md": "---\nname: override-skill\ntargets: [claude, cursor]\nmetadata:\n  targets: [claude]\n---\n# Metadata overrides",
	})

	claudePath := sb.CreateTarget("claude")
	cursorPath := sb.CreateTarget("cursor")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    path: ` + claudePath + `
  cursor:
    path: ` + cursorPath + `
`)

	result := sb.RunCLI("sync")
	result.AssertSuccess(t)

	// Declared targets remain metadata only, so sync still reaches every configured target.
	if !sb.IsSymlink(filepath.Join(claudePath, "override-skill")) {
		t.Error("override-skill should be in claude")
	}
	if !sb.IsSymlink(filepath.Join(cursorPath, "override-skill")) {
		t.Error("override-skill should also be in cursor")
	}
}
