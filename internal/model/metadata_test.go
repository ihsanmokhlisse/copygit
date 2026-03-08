package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultMetadata(t *testing.T) {
	m := DefaultMetadata()
	assert.Equal(t, VisibilityPrivate, m.Visibility)
	assert.Equal(t, "", m.Description)
	assert.True(t, m.WikiEnabled)
	assert.True(t, m.IssuesEnabled)
	assert.False(t, m.Archived)
}

func TestApplyOverrides(t *testing.T) {
	m := &RepoMetadata{
		Visibility:    VisibilityPrivate,
		Description:   "Original",
		WikiEnabled:   true,
	}

	newVis := VisibilityPublic
	newDesc := "Updated"
	newWiki := false

	overrides := &MetadataOverrides{
		Visibility:  &newVis,
		Description: &newDesc,
		WikiEnabled: &newWiki,
	}

	m.ApplyOverrides(overrides)

	assert.Equal(t, VisibilityPublic, m.Visibility)
	assert.Equal(t, "Updated", m.Description)
	assert.False(t, m.WikiEnabled)
}

func TestApplyOverridesNil(t *testing.T) {
	m := &RepoMetadata{Visibility: VisibilityPublic}
	m.ApplyOverrides(nil)
	assert.Equal(t, VisibilityPublic, m.Visibility) // unchanged
}
