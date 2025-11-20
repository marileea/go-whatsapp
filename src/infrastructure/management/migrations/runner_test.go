package migrations

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMigrationFilesEmbedded(t *testing.T) {
    entries, err := migrationFiles.ReadDir(".")
    require.NoError(t, err)
    
    foundUp := false
    foundDown := false
    
    for _, entry := range entries {
        if entry.Name() == "001_init_management.up.sql" {
            foundUp = true
        }
        if entry.Name() == "001_init_management.down.sql" {
            foundDown = true
        }
    }
    
    assert.True(t, foundUp, "Migration up file should be embedded")
    assert.True(t, foundDown, "Migration down file should be embedded")
}

func TestMigrationFileContent(t *testing.T) {
    content, err := migrationFiles.ReadFile("001_init_management.up.sql")
    require.NoError(t, err)
    assert.Contains(t, string(content), "CREATE TABLE IF NOT EXISTS tenants")
    assert.Contains(t, string(content), "CREATE TABLE IF NOT EXISTS api_keys")
    assert.Contains(t, string(content), "CREATE TABLE IF NOT EXISTS server_nodes")
}
