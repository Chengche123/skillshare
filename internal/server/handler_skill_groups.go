package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"skillshare/internal/install"
	ssync "skillshare/internal/sync"
)

type setSkillGroupsRequest struct {
	Groups *[]string `json:"groups"`
}

func (s *Server) handleSetSkillGroups(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if kind := r.URL.Query().Get("kind"); kind == "agent" {
		writeError(w, http.StatusBadRequest, "custom groups only support skills")
		return
	} else if kind != "" && kind != "skill" {
		writeError(w, http.StatusBadRequest, "invalid kind: "+kind)
		return
	}

	name := r.PathValue("name")
	var req setSkillGroupsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Groups == nil {
		writeError(w, http.StatusBadRequest, "groups is required")
		return
	}

	groups, err := install.NormalizeCustomGroups(*req.Groups)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.mu.RLock()
	source := s.skillsSource()
	s.mu.RUnlock()

	discovered, err := ssync.DiscoverSourceSkillsAll(source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to discover skills: "+err.Error())
		return
	}

	var match *ssync.DiscoveredSkill
	for i := range discovered {
		if discovered[i].FlatName == name {
			match = &discovered[i]
			break
		}
	}
	if match == nil {
		for i := range discovered {
			if filepath.Base(discovered[i].SourcePath) == name {
				match = &discovered[i]
				break
			}
		}
	}
	if match == nil {
		writeError(w, http.StatusNotFound, "resource not found: "+name)
		return
	}

	relPath := filepath.ToSlash(match.RelPath)

	s.mu.Lock()
	store, err := install.LoadMetadata(source)
	if err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "failed to load metadata: "+err.Error())
		return
	}

	entry := resolveSkillGroupMetadataEntry(store, relPath, discovered)

	if len(groups) == 0 {
		if entry != nil {
			entry.CustomGroups = nil
			if entry.HasMetadataBeyondCustomGroups() {
				store.Set(relPath, entry)
			} else {
				store.Remove(relPath)
			}
		}
	} else {
		if entry == nil {
			entry = &install.MetadataEntry{}
		}
		entry.CustomGroups = groups
		store.Set(relPath, entry)
	}

	if err := store.Save(source); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "failed to save metadata: "+err.Error())
		return
	}

	s.skillsStore = store
	s.mu.Unlock()

	s.writeOpsLog("set-skill-groups", "ok", start, map[string]any{
		"name":   name,
		"groups": groups,
		"scope":  "ui",
	}, "")

	writeJSON(w, map[string]any{"success": true})
}

func resolveSkillGroupMetadataEntry(store *install.MetadataStore, relPath string, discovered []ssync.DiscoveredSkill) *install.MetadataEntry {
	entry := store.GetByPathForCandidates(relPath, discoveredSkillRelPaths(discovered))
	if entry == nil {
		return nil
	}
	if store.MigrateLegacyKey(relPath, entry) {
		return store.Get(relPath)
	}
	return entry
}
