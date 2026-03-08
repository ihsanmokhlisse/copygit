package sync

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/provider"
)

func TestDetermineSourceProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := NewMetadataResolver(logger, provider.NewRegistry())

	tests := []struct {
		name         string
		globalConfig *model.RepoConfigMetadata
		allTargets   []model.RepoSyncTargetWithOverrides
		wantSource   string
	}{
		{
			name: "explicit inherit_from",
			globalConfig: &model.RepoConfigMetadata{
				InheritFrom: "gitlab",
			},
			allTargets: []model.RepoSyncTargetWithOverrides{
				{ProviderName: "github"},
				{ProviderName: "gitlab"},
			},
			wantSource: "gitlab",
		},
		{
			name:         "no config, use order",
			globalConfig: nil,
			allTargets: []model.RepoSyncTargetWithOverrides{
				{ProviderName: "gitlab"},
				{ProviderName: "github"},
			},
			wantSource: "github",
		},
		{
			name: "inherit_from=none",
			globalConfig: &model.RepoConfigMetadata{
				InheritFrom: "none",
			},
			allTargets: []model.RepoSyncTargetWithOverrides{
				{ProviderName: "github"},
			},
			wantSource: "github",  // Falls through since "none" is explicit rejection, not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.determineSourceProvider(nil, tt.globalConfig, tt.allTargets)
			assert.Equal(t, tt.wantSource, got)
		})
	}
}

func TestApplyGlobalOverrides(t *testing.T) {
	meta := &model.RepoMetadata{
		Visibility:    model.VisibilityPrivate,
		Description:   "Original",
		WikiEnabled:   true,
	}

	globalConfig := &model.RepoConfigMetadata{
		Visibility:  "public",
		Description: "Updated",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	resolver := NewMetadataResolver(logger, provider.NewRegistry())
	resolver.applyGlobalOverrides(meta, globalConfig)

	assert.Equal(t, model.VisibilityPublic, meta.Visibility)
	assert.Equal(t, "Updated", meta.Description)
	assert.True(t, meta.WikiEnabled)
}
