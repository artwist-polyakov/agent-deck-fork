package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestDB_CreatesIsolatedDB(t *testing.T) {
	db := NewTestDB(t)

	// Migrate should have already been called by NewTestDB.
	// Verify round-trip: save an instance and load it back.
	builder := NewInstanceBuilder("test-1", "Test Session")
	err := db.SaveInstances(builder.BuildSlice())
	require.NoError(t, err, "SaveInstances should succeed")

	loaded, err := db.LoadInstances()
	require.NoError(t, err, "LoadInstances should succeed")
	require.Len(t, loaded, 1, "should load exactly 1 instance")
	assert.Equal(t, "test-1", loaded[0].ID)
	assert.Equal(t, "Test Session", loaded[0].Title)
}

func TestNewTestDB_IsolationBetweenTests(t *testing.T) {
	db1 := NewTestDB(t)
	db2 := NewTestDB(t)

	// Save to db1.
	builder := NewInstanceBuilder("iso-1", "Isolated")
	err := db1.SaveInstances(builder.BuildSlice())
	require.NoError(t, err)

	// db2 should be empty (independent database).
	loaded, err := db2.LoadInstances()
	require.NoError(t, err)
	assert.Empty(t, loaded, "db2 should be empty; databases must be independent")

	// db1 should have the saved instance.
	loaded1, err := db1.LoadInstances()
	require.NoError(t, err)
	assert.Len(t, loaded1, 1, "db1 should have 1 instance")
}

func TestInstanceBuilder_DefaultValues(t *testing.T) {
	row := NewInstanceBuilder("id-1", "My Title").Build()

	assert.Equal(t, "id-1", row.ID)
	assert.Equal(t, "My Title", row.Title)
	assert.Equal(t, "shell", row.Tool, "default tool should be shell")
	assert.Equal(t, "idle", row.Status, "default status should be idle")
	assert.Equal(t, "/tmp/test", row.ProjectPath, "default project path")
	assert.Equal(t, "test-group", row.GroupPath, "default group path")
	assert.False(t, row.CreatedAt.IsZero(), "CreatedAt should be set")
}

func TestInstanceBuilder_WithMethods(t *testing.T) {
	row := NewInstanceBuilder("id-2", "Built").
		WithTool("claude").
		WithStatus("running").
		WithParent("parent-1").
		WithProject("/home/user/project").
		WithGroup("my-group").
		WithCommand("echo hello").
		Build()

	assert.Equal(t, "claude", row.Tool)
	assert.Equal(t, "running", row.Status)
	assert.Equal(t, "parent-1", row.ParentSessionID)
	assert.Equal(t, "/home/user/project", row.ProjectPath)
	assert.Equal(t, "my-group", row.GroupPath)
	assert.Equal(t, "echo hello", row.Command)
}

func TestInstanceBuilder_SaveAndLoad(t *testing.T) {
	db := NewTestDB(t)

	builder := NewInstanceBuilder("save-1", "Persist Me").
		WithTool("claude").
		WithStatus("running").
		WithParent("parent-x")

	err := db.SaveInstances(builder.BuildSlice())
	require.NoError(t, err)

	loaded, err := db.LoadInstances()
	require.NoError(t, err)
	require.Len(t, loaded, 1)

	got := loaded[0]
	assert.Equal(t, "save-1", got.ID)
	assert.Equal(t, "Persist Me", got.Title)
	assert.Equal(t, "claude", got.Tool)
	assert.Equal(t, "running", got.Status)
	assert.Equal(t, "parent-x", got.ParentSessionID)
}
