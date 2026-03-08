package sync

import (
	"context"
	"fmt"
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

// CheckUnsupportedFields identifies metadata fields not supported by a provider.
func CheckUnsupportedFields(meta *model.RepoMetadata, providerType string) []string {
	var unsupported []string

	switch providerType {
	case "gitlab":
		if meta.Language != "" {
			unsupported = append(unsupported, "language")
		}
	case "gitea":
		if meta.Language != "" {
			unsupported = append(unsupported, "language")
		}
		if meta.License != "" {
			unsupported = append(unsupported, "license")
		}
		if meta.Homepage != "" {
			unsupported = append(unsupported, "homepage")
		}
		if meta.WikiEnabled {
			unsupported = append(unsupported, "wiki_enabled")
		}
		if meta.IssuesEnabled {
			unsupported = append(unsupported, "issues_enabled")
		}
	}

	return unsupported
}

// LogMetadataSync logs metadata synchronization results.
func LogMetadataSync(
	ctx context.Context,
	logger *slog.Logger,
	providerName string,
	sourceProvider string,
	hasGlobalOverrides bool,
	hasTargetOverrides bool,
	unsupportedFields []string,
) {
	if sourceProvider != "" {
		logger.InfoContext(ctx,
			"inherited metadata",
			"provider", providerName,
			"from", sourceProvider,
		)
	} else {
		logger.InfoContext(ctx,
			"using default metadata",
			"provider", providerName,
		)
	}

	if hasGlobalOverrides {
		logger.DebugContext(ctx,
			"applied global metadata overrides",
			"provider", providerName,
		)
	}

	if hasTargetOverrides {
		logger.DebugContext(ctx,
			"applied provider-specific metadata overrides",
			"provider", providerName,
		)
	}

	for _, field := range unsupportedFields {
		logger.WarnContext(ctx,
			fmt.Sprintf("provider does not support %s field", field),
			"provider", providerName,
		)
	}
}
