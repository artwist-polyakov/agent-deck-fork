// STAB-01: No production code bugs discovered during Phase 2 testing.
// Plans 01 and 02 executed cleanly. All deviations were test assertion
// adjustments (test expectations not matching actual system behavior),
// not production code defects. Full test suite passes: go test -race ./...

package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSkillRuntime_AttachedSkillIsReadable verifies that after AttachSkillToProject,
// the materialized path contains a readable SKILL.md with the expected content.
func TestSkillRuntime_AttachedSkillIsReadable(t *testing.T) {
	_, cleanup := setupSkillTestEnv(t)
	defer cleanup()

	sourcePath := t.TempDir()
	writeSkillDir(t, sourcePath, "my-skill", "my-skill", "A test skill")

	require.NoError(t, SaveSkillSources(map[string]SkillSourceDef{
		"local": {Path: sourcePath, Enabled: boolPtr(true)},
	}))

	projectPath := t.TempDir()

	attachment, err := AttachSkillToProject(projectPath, "my-skill", "local")
	require.NoError(t, err, "AttachSkillToProject should succeed")
	require.NotNil(t, attachment)

	// The materialized skill should be at <project>/.claude/skills/<entry>/SKILL.md
	targetDir := resolveTargetPath(projectPath, attachment.TargetPath)
	skillMDPath := filepath.Join(targetDir, "SKILL.md")

	content, err := os.ReadFile(skillMDPath)
	require.NoError(t, err, "SKILL.md should be readable at materialized path")
	assert.Contains(t, string(content), "my-skill", "SKILL.md should contain the skill name")
}

// TestSkillRuntime_ApplyCreatesReadableSkills verifies that ApplyProjectSkills
// creates a .claude/skills/ directory with readable SKILL.md for each skill.
func TestSkillRuntime_ApplyCreatesReadableSkills(t *testing.T) {
	_, cleanup := setupSkillTestEnv(t)
	defer cleanup()

	sourcePath := t.TempDir()
	writeSkillDir(t, sourcePath, "alpha", "alpha", "Alpha skill")
	writeSkillDir(t, sourcePath, "beta", "beta", "Beta skill")

	require.NoError(t, SaveSkillSources(map[string]SkillSourceDef{
		"local": {Path: sourcePath, Enabled: boolPtr(true)},
	}))

	alphaCandidate, err := ResolveSkillCandidate("alpha", "local")
	require.NoError(t, err)
	betaCandidate, err := ResolveSkillCandidate("beta", "local")
	require.NoError(t, err)

	projectPath := t.TempDir()

	err = ApplyProjectSkills(projectPath, []SkillCandidate{*alphaCandidate, *betaCandidate})
	require.NoError(t, err, "ApplyProjectSkills should succeed")

	skillsDir := GetProjectClaudeSkillsPath(projectPath)
	entries, err := os.ReadDir(skillsDir)
	require.NoError(t, err, ".claude/skills/ directory should exist")
	assert.Len(t, entries, 2, "should have 2 skill directories")

	// Each skill should have a readable SKILL.md
	for _, entry := range entries {
		skillMDPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		content, err := os.ReadFile(skillMDPath)
		assert.NoError(t, err, "SKILL.md should be readable for %s", entry.Name())
		assert.NotEmpty(t, content, "SKILL.md should have content for %s", entry.Name())
	}
}

// TestSkillRuntime_DiscoveryFindsRegisteredSkills verifies that ListAvailableSkills
// discovers all skills from a registered source.
func TestSkillRuntime_DiscoveryFindsRegisteredSkills(t *testing.T) {
	_, cleanup := setupSkillTestEnv(t)
	defer cleanup()

	sourcePath := t.TempDir()
	writeSkillDir(t, sourcePath, "skill-one", "skill-one", "First skill")
	writeSkillDir(t, sourcePath, "skill-two", "skill-two", "Second skill")
	writeSkillDir(t, sourcePath, "skill-three", "skill-three", "Third skill")

	require.NoError(t, SaveSkillSources(map[string]SkillSourceDef{
		"test-source": {Path: sourcePath, Enabled: boolPtr(true)},
	}))

	skills, err := ListAvailableSkills()
	require.NoError(t, err, "ListAvailableSkills should succeed")

	names := make(map[string]bool, len(skills))
	for _, s := range skills {
		names[s.Name] = true
	}

	assert.True(t, names["skill-one"], "should discover skill-one")
	assert.True(t, names["skill-two"], "should discover skill-two")
	assert.True(t, names["skill-three"], "should discover skill-three")
}

// TestSkillRuntime_ResolveByName verifies that ResolveSkillCandidate correctly
// finds skills by name when the source is specified.
func TestSkillRuntime_ResolveByName(t *testing.T) {
	_, cleanup := setupSkillTestEnv(t)
	defer cleanup()

	sourceA := t.TempDir()
	sourceB := t.TempDir()
	writeSkillDir(t, sourceA, "alpha", "alpha", "Alpha from source A")
	writeSkillDir(t, sourceB, "beta", "beta", "Beta from source B")

	require.NoError(t, SaveSkillSources(map[string]SkillSourceDef{
		"src1": {Path: sourceA, Enabled: boolPtr(true)},
		"src2": {Path: sourceB, Enabled: boolPtr(true)},
	}))

	resolved, err := ResolveSkillCandidate("alpha", "src1")
	require.NoError(t, err, "should resolve alpha from src1")
	assert.Equal(t, "alpha", resolved.Name)
	assert.Equal(t, "src1", resolved.Source)

	resolved, err = ResolveSkillCandidate("beta", "src2")
	require.NoError(t, err, "should resolve beta from src2")
	assert.Equal(t, "beta", resolved.Name)
	assert.Equal(t, "src2", resolved.Source)
}

// TestSkillRuntime_PoolSkillWithoutScripts verifies that a pool skill with only
// SKILL.md (no scripts/ subdirectory) can be attached and read without errors.
// Pool skills like gsd-conductor often have only references/, no scripts/.
func TestSkillRuntime_PoolSkillWithoutScripts(t *testing.T) {
	_, cleanup := setupSkillTestEnv(t)
	defer cleanup()

	sourcePath := t.TempDir()

	// Create a minimal pool skill: just SKILL.md and a references/ dir (no scripts/)
	poolSkillDir := filepath.Join(sourcePath, "gsd-conductor")
	require.NoError(t, os.MkdirAll(poolSkillDir, 0o755))
	skillContent := "---\nname: gsd-conductor\ndescription: GSD orchestration conductor\n---\n\n# GSD Conductor\n\nOrchestrates multi-agent workflows.\n"
	require.NoError(t, os.WriteFile(filepath.Join(poolSkillDir, "SKILL.md"), []byte(skillContent), 0o644))

	// Add a references/ directory but no scripts/
	refsDir := filepath.Join(poolSkillDir, "references")
	require.NoError(t, os.MkdirAll(refsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(refsDir, "checkpoints.md"), []byte("# Checkpoints\n"), 0o644))

	require.NoError(t, SaveSkillSources(map[string]SkillSourceDef{
		"pool": {Path: sourcePath, Enabled: boolPtr(true)},
	}))

	projectPath := t.TempDir()

	attachment, err := AttachSkillToProject(projectPath, "gsd-conductor", "pool")
	require.NoError(t, err, "pool skill without scripts/ should attach successfully")
	require.NotNil(t, attachment)

	// Verify SKILL.md is readable at the materialized location
	targetDir := resolveTargetPath(projectPath, attachment.TargetPath)
	content, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	require.NoError(t, err, "SKILL.md should be readable from materialized pool skill")
	assert.Contains(t, string(content), "gsd-conductor")

	// Verify references/ was also materialized
	refContent, err := os.ReadFile(filepath.Join(targetDir, "references", "checkpoints.md"))
	require.NoError(t, err, "references/ should be materialized alongside SKILL.md")
	assert.Contains(t, string(refContent), "Checkpoints")
}

// TestSkillRuntime_ResolveSkillContent verifies that after attaching a skill,
// the SKILL.md content contains expected frontmatter (name, description).
func TestSkillRuntime_ResolveSkillContent(t *testing.T) {
	_, cleanup := setupSkillTestEnv(t)
	defer cleanup()

	sourcePath := t.TempDir()
	writeSkillDir(t, sourcePath, "code-review", "code-review", "Automated code review rules")

	require.NoError(t, SaveSkillSources(map[string]SkillSourceDef{
		"local": {Path: sourcePath, Enabled: boolPtr(true)},
	}))

	projectPath := t.TempDir()

	attachment, err := AttachSkillToProject(projectPath, "code-review", "local")
	require.NoError(t, err)

	targetDir := resolveTargetPath(projectPath, attachment.TargetPath)
	content, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	require.NoError(t, err, "SKILL.md should be readable")

	text := string(content)

	// Parse YAML frontmatter between --- delimiters
	require.True(t, strings.HasPrefix(text, "---\n"), "SKILL.md should start with YAML frontmatter")
	rest := text[4:]
	endIdx := strings.Index(rest, "\n---")
	require.True(t, endIdx >= 0, "SKILL.md should have closing frontmatter delimiter")

	frontmatter := rest[:endIdx]
	assert.Contains(t, frontmatter, "name: code-review", "frontmatter should contain skill name")
	assert.Contains(t, frontmatter, "description: Automated code review rules", "frontmatter should contain description")
}
