package sync

import (
	"context"
	"log/slog"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/provider"
)

// MetadataResolver discovers and applies metadata inheritance rules.
type MetadataResolver struct {
	logger      *slog.Logger
	providerReg *provider.Registry
}

// NewMetadataResolver creates a new metadata resolver.
func NewMetadataResolver(
	logger *slog.Logger,
	providerReg *provider.Registry,
) *MetadataResolver {
	return &MetadataResolver{
		logger:      logger,
		providerReg: providerReg,
	}
}

// ResolveMetadata discovers source metadata and applies overrides.
func (r *MetadataResolver) ResolveMetadata(
	ctx context.Context,
	globalConfig *model.RepoConfigMetadata,
	target model.RepoSyncTargetWithOverrides,
	allTargets []model.RepoSyncTargetWithOverrides,
) (*model.RepoMetadata, error) {
	// Step 1: Determine source provider
	sourceProvider := r.determineSourceProvider(ctx, globalConfig, allTargets)
	if sourceProvider == "" {
		// No source found, use defaults
		return model.DefaultMetadata(), nil
	}

	// Step 2: Fall back to defaults (actual implementation deferred)
	meta := model.DefaultMetadata()

	// Step 3: Apply global overrides
	if globalConfig != nil {
		r.applyGlobalOverrides(meta, globalConfig)
	}

	// Step 4: Apply target-specific overrides
	if target.Overrides != nil {
		meta.ApplyOverrides(target.Overrides)
	}

	return meta, nil
}

// determineSourceProvider finds which provider to inherit metadata from.
func (r *MetadataResolver) determineSourceProvider(
	ctx context.Context,
	globalConfig *model.RepoConfigMetadata,
	allTargets []model.RepoSyncTargetWithOverrides,
) string {
	if globalConfig != nil && globalConfig.InheritFrom != "" && globalConfig.InheritFrom != "none" {
		return globalConfig.InheritFrom
	}

	// Find first provider in order
	order := []string{"github", "gitlab", "gitea"}
	for _, name := range order {
		for _, target := range allTargets {
			if target.ProviderName == name {
				return name
			}
		}
	}

	return ""
}

// applyGlobalOverrides applies config-level metadata overrides.
func (r *MetadataResolver) applyGlobalOverrides(
	meta *model.RepoMetadata,
	globalConfig *model.RepoConfigMetadata,
) {
	if globalConfig.Visibility != "" {
		vis := model.Visibility(globalConfig.Visibility)
		meta.Visibility = vis
	}
	if globalConfig.Description != "" {
		meta.Description = globalConfig.Description
	}
	if globalConfig.Homepage != "" {
		meta.Homepage = globalConfig.Homepage
	}
	if len(globalConfig.Topics) > 0 {
		meta.Topics = globalConfig.Topics
	}
	if globalConfig.WikiEnabled != nil {
		meta.WikiEnabled = *globalConfig.WikiEnabled
	}
	if globalConfig.IssuesEnabled != nil {
		meta.IssuesEnabled = *globalConfig.IssuesEnabled
	}
	if globalConfig.Archived != nil {
		meta.Archived = *globalConfig.Archived
	}
}
