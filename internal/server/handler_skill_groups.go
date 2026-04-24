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
	Groups []string `json:"groups"`
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

	groups, err := install.NormalizeCustomGroups(req.Groups)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.mu.RLock()
	source := s.cfg.Source
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

	store, err := install.LoadMetadata(source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load metadata: "+err.Error())
		return
	}

	relPath := filepath.ToSlash(match.RelPath)
	entry := store.GetByPath(relPath)
	if entry != nil {
		store.MigrateLegacyKey(relPath, entry)
		entry = store.Get(relPath)
	}

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
		writeError(w, http.StatusInternalServerError, "failed to save metadata: "+err.Error())
		return
	}

	s.mu.Lock()
	s.skillsStore = store
	s.mu.Unlock()

	s.writeOpsLog("set-skill-groups", "ok", start, map[string]any{
		"name":   name,
		"groups": groups,
		"scope":  "ui",
	}, "")

	writeJSON(w, map[string]any{"success": true})
}
