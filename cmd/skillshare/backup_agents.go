package main

import (
	"fmt"

	"skillshare/internal/backup"
	"skillshare/internal/config"
	"skillshare/internal/ui"
)

// createAgentBackup backs up agent target directories.
// Agent backups use "<target>-agents" as the backup entry name.
func createAgentBackup(targetName string, dryRun bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	builtinAgents := config.DefaultAgentTargets()
	ui.Header("Creating agent backup")
	if dryRun {
		ui.Warning("Dry run mode - no backups will be created")
	}

	created := 0
	for name := range cfg.Targets {
		if targetName != "" && name != targetName {
			continue
		}

		agentPath := resolveAgentTargetPath(cfg.Targets[name], builtinAgents, name)
		if agentPath == "" {
			continue
		}

		entryName := name + "-agents"

		if dryRun {
			ui.Info("%s: would backup agents from %s", entryName, agentPath)
			continue
		}

		backupPath, backupErr := backup.Create(entryName, agentPath)
		if backupErr != nil {
			ui.Warning("Failed to backup %s: %v", entryName, backupErr)
			continue
		}
		if backupPath != "" {
			ui.StepDone(entryName, backupPath)
			created++
		} else {
			ui.StepSkip(entryName, "nothing to backup")
		}
	}

	if created == 0 && !dryRun {
		ui.Info("No agent targets to backup")
	}

	return nil
}

// restoreAgentBackup restores agent target directories from backup.
func restoreAgentBackup(targetName, fromTimestamp string, force, dryRun bool) error {
	if targetName == "" {
		return fmt.Errorf("usage: skillshare restore agents <target> [--from <timestamp>] [--force] [--dry-run]")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	builtinAgents := config.DefaultAgentTargets()
	agentPath := resolveAgentTargetPath(cfg.Targets[targetName], builtinAgents, targetName)
	if agentPath == "" {
		return fmt.Errorf("target '%s' has no agent path configured", targetName)
	}

	entryName := targetName + "-agents"
	ui.Header(fmt.Sprintf("Restoring agents for %s", targetName))

	if dryRun {
		ui.Warning("Dry run mode - no changes will be made")
		ui.Info("Would restore %s to %s", entryName, agentPath)
		return nil
	}

	opts := backup.RestoreOptions{Force: force}
	if fromTimestamp != "" {
		return restoreFromTimestamp(entryName, agentPath, fromTimestamp, opts)
	}
	return restoreFromLatest(entryName, agentPath, opts)
}
