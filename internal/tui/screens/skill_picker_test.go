package screens

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/internal/components/skills"
	"github.com/gentleman-programming/gentle-ai/internal/tui/styles"
)

// TestSkillPickerGroupsCoverAllNavigableSkills verifies that the two rendered
// groups (SDD + Foundation) exactly cover AllSkillsOrdered(), which is what
// navigation and toggling use.
//
// Regression test: the picker used to keep private copies of the group lists.
// When skill-registry was added to the catalog, those copies drifted: the
// cursor could land on a 16th row that rendered nothing, and pressing
// space/enter on that invisible row silently toggled skill-registry.
func TestSkillPickerGroupsCoverAllNavigableSkills(t *testing.T) {
	all := AllSkillsOrdered()
	grouped := append(skills.SDDSkillIDs(), skills.FoundationSkillIDs()...)

	if len(grouped) != len(all) {
		t.Fatalf("rendered groups cover %d skills but %d are navigable — the picker has %d phantom row(s)",
			len(grouped), len(all), len(all)-len(grouped))
	}
	for i, id := range all {
		if grouped[i] != id {
			t.Errorf("row %d: rendered group order has %q, navigation order has %q", i, grouped[i], id)
		}
	}
}

// TestSkillPickerEverySkillHasExplicitLabel verifies that every navigable
// skill has a human-readable label instead of falling back to its raw ID.
func TestSkillPickerEverySkillHasExplicitLabel(t *testing.T) {
	for _, id := range AllSkillsOrdered() {
		if _, ok := skillLabels[id]; !ok {
			t.Errorf("skill %q has no entry in skillLabels — it would render as its raw ID", id)
		}
	}
}

// TestSkillPickerCursorIsAlwaysVisible verifies that for every navigable row
// (each skill plus the Continue/Back actions) the rendered output shows the
// cursor marker exactly once.
func TestSkillPickerCursorIsAlwaysVisible(t *testing.T) {
	all := AllSkillsOrdered()
	total := SkillPickerOptionCount()

	if want := len(all) + len(SkillPickerOptions()); total != want {
		t.Fatalf("SkillPickerOptionCount() = %d, want %d", total, want)
	}

	for cursor := 0; cursor < total; cursor++ {
		out := RenderSkillPicker(all, cursor)
		if got := strings.Count(out, styles.Cursor); got != 1 {
			t.Errorf("cursor=%d: rendered output shows the cursor marker %d times, want exactly 1", cursor, got)
		}
	}
}

// TestSkillPickerRendersSkillRegistryRow verifies that the skill-registry row
// is visible and highlighted when the cursor is on the last skill.
func TestSkillPickerRendersSkillRegistryRow(t *testing.T) {
	all := AllSkillsOrdered()
	out := RenderSkillPicker(all, len(all)-1)

	if !strings.Contains(out, "Skill Registry") {
		t.Fatalf("rendered picker does not show the %q row:\n%s", "Skill Registry", out)
	}

	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, styles.Cursor) {
			if !strings.Contains(line, "Skill Registry") {
				t.Errorf("cursor on last skill should highlight %q; highlighted line: %q", "Skill Registry", line)
			}
			return
		}
	}
	t.Fatalf("no line contains the cursor marker")
}
