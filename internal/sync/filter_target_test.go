package sync

import "testing"

func TestFilterSkillsByTarget_NilPassesThrough(t *testing.T) {
	skills := []DiscoveredSkill{
		{FlatName: "no-targets", Targets: nil},
	}

	result := FilterSkillsByTarget(skills, "claude")
	if len(result) != 1 {
		t.Errorf("nil Targets should pass through, got %d", len(result))
	}
}

func TestFilterSkillsByTarget_DeclaredTargetsAreIgnored(t *testing.T) {
	skills := []DiscoveredSkill{
		{FlatName: "claude-only", Targets: []string{"claude"}},
		{FlatName: "cursor-only", Targets: []string{"cursor"}},
	}

	result := FilterSkillsByTarget(skills, "claude")
	if len(result) != 2 {
		t.Fatalf("expected both skills to pass through, got %d", len(result))
	}
	if result[0].FlatName != "claude-only" || result[1].FlatName != "cursor-only" {
		t.Fatalf("unexpected result order/content: %+v", result)
	}
}

func TestFilterSkillsByTarget_MixedDeclaredTargetsStillPassThrough(t *testing.T) {
	skills := []DiscoveredSkill{
		{FlatName: "all-targets", Targets: nil},
		{FlatName: "claude-only", Targets: []string{"claude"}},
		{FlatName: "cursor-only", Targets: []string{"cursor"}},
		{FlatName: "multi", Targets: []string{"claude", "cursor"}},
	}

	result := FilterSkillsByTarget(skills, "claude")
	if len(result) != len(skills) {
		t.Fatalf("expected all %d skills to pass through, got %d", len(skills), len(result))
	}
	for i := range skills {
		if result[i].FlatName != skills[i].FlatName {
			t.Fatalf("result[%d] = %q, want %q", i, result[i].FlatName, skills[i].FlatName)
		}
	}
}

func TestFilterSkillsByTarget_PathTargetsRestrictSync(t *testing.T) {
	skills := []DiscoveredSkill{
		{FlatName: "shared"},
		{FlatName: "cursor-only", Targets: []string{"cursor"}, PathTargets: []string{"cursor"}},
		{FlatName: "droid-only", Targets: []string{"droid"}, PathTargets: []string{"droid"}},
	}

	result := FilterSkillsByTarget(skills, "cursor")
	if len(result) != 2 {
		t.Fatalf("expected shared and cursor path skills, got %d", len(result))
	}
	if result[0].FlatName != "shared" || result[1].FlatName != "cursor-only" {
		t.Fatalf("unexpected result order/content: %+v", result)
	}
}

func TestFilterSkillsByTarget_PathTargetsOverrideDeclaredMetadata(t *testing.T) {
	skills := []DiscoveredSkill{
		{
			FlatName:    "cursor-hosted",
			Targets:     []string{"droid"},
			PathTargets: []string{"cursor"},
		},
	}

	if result := FilterSkillsByTarget(skills, "cursor"); len(result) != 1 {
		t.Fatalf("expected path-inferred cursor target to pass, got %d", len(result))
	}
	if result := FilterSkillsByTarget(skills, "droid"); len(result) != 0 {
		t.Fatalf("declared target should not override path inference, got %d", len(result))
	}
}
