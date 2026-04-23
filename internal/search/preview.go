package search

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	ghclient "skillshare/internal/github"
)

// SkillPreview contains the full SKILL.md content + parsed frontmatter metadata
// for previewing a remote skill before installation.
type SkillPreview struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	License     string   `json:"license,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Content     string   `json:"content"`
	Source      string   `json:"source"`
	Stars       int      `json:"stars"`
	Owner       string   `json:"owner"`
	Repo        string   `json:"repo"`
}

// FetchSkillContent fetches the full SKILL.md from a GitHub repository and
// returns parsed frontmatter metadata along with the raw content.
// The path parameter is the subdirectory within the repo (empty or "." for root).
func FetchSkillContent(client *http.Client, owner, repo, path string) (*SkillPreview, error) {
	skillPath := "SKILL.md"
	if path != "" && path != "." {
		skillPath = path + "/SKILL.md"
	}

	apiURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/%s",
		owner, repo, url.PathEscape(skillPath),
	)

	req, err := ghclient.NewRequest(apiURL)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := ghclient.CheckRateLimit(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("SKILL.md not found at %s/%s/%s", owner, repo, path)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var content gitHubContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, err
	}
	if content.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s", content.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return nil, err
	}

	body := string(decoded)

	// Build source string
	source := owner + "/" + repo
	if path != "" && path != "." {
		source = source + "/" + path
	}

	preview := &SkillPreview{
		Name:        parseFrontmatterField(body, "name"),
		Description: parseFrontmatterField(body, "description"),
		License:     parseFrontmatterField(body, "license"),
		Content:     body,
		Source:      source,
		Owner:       owner,
		Repo:        repo,
	}

	// Parse tags (comma-separated or YAML list on one line)
	if tagsRaw := parseFrontmatterField(body, "tags"); tagsRaw != "" {
		tagsRaw = strings.Trim(tagsRaw, "[]")
		for _, t := range strings.Split(tagsRaw, ",") {
			t = strings.TrimSpace(t)
			t = strings.Trim(t, `"'`)
			if t != "" {
				preview.Tags = append(preview.Tags, t)
			}
		}
	}

	// Fetch star count (best-effort, don't fail on error)
	if stars, err := fetchRepoStars(client, owner, repo); err == nil {
		preview.Stars = stars
	}

	return preview, nil
}
