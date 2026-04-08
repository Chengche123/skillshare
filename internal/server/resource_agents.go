package server

import (
	"fmt"
	"path/filepath"
	"strings"

	"skillshare/internal/resource"
)

func agentDisplayName(relPath string) string {
	return strings.TrimSuffix(relPath, ".md")
}

func matchesAgentName(d resource.DiscoveredResource, name string) bool {
	return d.FlatName == name ||
		d.Name == name ||
		d.RelPath == name ||
		agentDisplayName(d.RelPath) == name
}

func resolveAgentResource(agentsSource, name string) (resource.DiscoveredResource, error) {
	discovered, err := resource.AgentKind{}.Discover(agentsSource)
	if err != nil {
		return resource.DiscoveredResource{}, fmt.Errorf("failed to discover agents: %w", err)
	}
	for _, d := range discovered {
		if matchesAgentName(d, name) {
			return d, nil
		}
	}
	return resource.DiscoveredResource{}, fmt.Errorf("agent not found: %s", name)
}

func (s *Server) resolveAgentRelPathWithStatus(agentsSource, name string) (string, bool, error) {
	discovered, err := resource.AgentKind{}.Discover(agentsSource)
	if err != nil {
		return "", false, fmt.Errorf("failed to discover agents: %w", err)
	}
	for _, d := range discovered {
		if matchesAgentName(d, name) {
			return d.RelPath, d.Disabled, nil
		}
	}
	return "", false, fmt.Errorf("agent not found: %s", name)
}

func agentMetaKey(relPath string) string {
	return strings.TrimSuffix(filepath.ToSlash(relPath), ".md")
}
