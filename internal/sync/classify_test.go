package sync

import "testing"

func TestClassifySkillForTarget(t *testing.T) {
	tests := []struct {
		name       string
		flatName   string
		include    []string
		exclude    []string
		wantStatus string
		wantReason string
	}{
		{name: "synced_no_filters", flatName: "my-skill", wantStatus: "synced"},
		{name: "excluded_by_pattern", flatName: "legacy-tool", exclude: []string{"legacy*"}, wantStatus: "excluded", wantReason: "legacy*"},
		{name: "not_included", flatName: "backend-api", include: []string{"frontend*"}, wantStatus: "not_included"},
		{name: "included_by_pattern", flatName: "frontend-design", include: []string{"frontend*"}, wantStatus: "synced"},
		{name: "exclude_wins_over_include", flatName: "frontend-legacy", include: []string{"frontend*"}, exclude: []string{"*legacy*"}, wantStatus: "excluded", wantReason: "*legacy*"},
		{name: "include_miss_beats_exclude_match", flatName: "legacy-tool", include: []string{"frontend*"}, exclude: []string{"legacy*"}, wantStatus: "not_included"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, reason := ClassifySkillForTarget(tt.flatName, tt.include, tt.exclude)
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}
