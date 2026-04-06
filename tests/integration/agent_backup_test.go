//go:build !online

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"skillshare/internal/testutil"
)

func TestBackup_Agents_CreatesBackup(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	createAgentSource(t, sb, map[string]string{
		"tutor.md": "# Tutor agent",
	})
	claudeAgents := createAgentTarget(t, sb, "claude")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    skills:
      path: ` + sb.CreateTarget("claude") + `
    agents:
      path: ` + claudeAgents + `
`)

	// Sync agents first so there's something to backup
	sb.RunCLI("sync", "agents")

	result := sb.RunCLI("backup", "agents")
	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "agent backup")
}

func TestBackup_Agents_DryRun(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	createAgentSource(t, sb, map[string]string{
		"tutor.md": "# Tutor agent",
	})
	claudeAgents := createAgentTarget(t, sb, "claude")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    skills:
      path: ` + sb.CreateTarget("claude") + `
    agents:
      path: ` + claudeAgents + `
`)

	sb.RunCLI("sync", "agents")

	result := sb.RunCLI("backup", "agents", "--dry-run")
	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "Dry run")
}

func TestBackup_Agents_RestoreRoundTrip(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	createAgentSource(t, sb, map[string]string{
		"tutor.md": "# Tutor agent",
	})
	claudeAgents := createAgentTarget(t, sb, "claude")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    skills:
      path: ` + sb.CreateTarget("claude") + `
    agents:
      path: ` + claudeAgents + `
`)

	// Sync then backup
	sb.RunCLI("sync", "agents")
	sb.RunCLI("backup", "agents")

	// Verify symlink exists
	linkPath := filepath.Join(claudeAgents, "tutor.md")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("expected agent symlink at %s", linkPath)
	}

	// Delete the agent from target
	os.Remove(linkPath)
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatal("symlink should be removed")
	}

	// Restore
	result := sb.RunCLI("restore", "agents", "claude", "--force")
	result.AssertSuccess(t)
	result.AssertAnyOutputContains(t, "Restored")
}

func TestBackup_Default_DoesNotBackupAgents(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.CreateSkill("my-skill", map[string]string{
		"SKILL.md": "---\nname: my-skill\n---\n# Content",
	})
	claudeSkills := sb.CreateTarget("claude")

	sb.WriteConfig(`source: ` + sb.SourcePath + `
targets:
  claude:
    skills:
      path: ` + claudeSkills + `
`)

	// Default backup should only backup skills, not mention agents
	result := sb.RunCLI("backup")
	result.AssertSuccess(t)
	result.AssertOutputNotContains(t, "agent")
}

func TestRestore_Agents_ProjectModeRejected(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	sb.WriteConfig(`source: ` + sb.SourcePath + "\ntargets: {}\n")

	result := sb.RunCLI("restore", "-p", "agents", "claude")
	result.AssertFailure(t)
	result.AssertAnyOutputContains(t, "not supported in project mode")
}
