package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/provider"
)

// Orchestrator coordinates sync operations across multiple providers.
// It is repo-agnostic; you provide providers and targets per call.
type Orchestrator struct {
	gitExec   git.GitExecutor
	remoteMgr *git.RemoteManager
	branchMgr *git.BranchManager
	credMgr   credential.Manager
	logger    *slog.Logger
}

// NewOrchestrator creates a new sync orchestrator.
func NewOrchestrator(gitExec git.GitExecutor, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{
		gitExec:   gitExec,
		remoteMgr: git.NewRemoteManager(gitExec, logger),
		branchMgr: git.NewBranchManager(gitExec, logger),
		credMgr:   nil,
		logger:    logger,
	}
}

// WithCredentialManager sets the credential manager for metadata operations.
func (o *Orchestrator) WithCredentialManager(credMgr credential.Manager) *Orchestrator {
	o.credMgr = credMgr
	return o
}

// SyncReport summarizes the results of a sync operation.
type SyncReport struct { //nolint:revive // established API name
	OperationType   model.OperationType   `json:"operation_type"`
	RepoPath        string                `json:"repo_path"`
	TotalTargets    int                   `json:"total_targets"`
	SuccessCount    int                   `json:"success_count"`
	FailureCount    int                   `json:"failure_count"`
	Operations      []model.SyncOperation `json:"operations"`
	StartTime       time.Time             `json:"start_time"`
	EndTime         time.Time             `json:"end_time"`
	DurationSeconds float64               `json:"duration_seconds"`
	ReposCreated    []string              `json:"repos_created,omitempty"`
	MetadataSynced  []string              `json:"metadata_synced,omitempty"`
	MetadataWarnings []string             `json:"metadata_warnings,omitempty"`
}

// Push synchronizes local repo to all specified providers.
func (o *Orchestrator) Push(
	ctx context.Context,
	repoPath string,
	providers map[string]provider.Provider,
	targets []model.RepoSyncTarget,
) (*SyncReport, error) {
	o.logger.InfoContext(ctx, "starting push", "repo", repoPath, "targets", len(targets))

	report := &SyncReport{
		OperationType: model.OpPush,
		RepoPath:      repoPath,
		TotalTargets:  len(targets),
		StartTime:     time.Now(),
		Operations:    []model.SyncOperation{},
	}

	// Get current branch
	currentBranch, err := o.branchMgr.CurrentBranch(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	for _, target := range targets {
		if !target.Enabled {
			continue
		}

		prov, ok := providers[target.ProviderName]
		if !ok {
			o.logger.WarnContext(ctx, "provider not found", "provider", target.ProviderName)
			report.FailureCount++
			continue
		}

		op := model.SyncOperation{
			ID:           fmt.Sprintf("%s-%s-%d", repoPath, target.ProviderName, time.Now().UnixNano()),
			ProviderName: target.ProviderName,
			RepoPath:     repoPath,
			Type:         model.OpPush,
			Branch:       currentBranch,
			Status:       model.StatusInProgress,
			CreatedAt:    time.Now(),
		}

		if err := o.executePush(ctx, repoPath, prov, target, currentBranch); err != nil {
			op.Status = model.StatusFailed
			op.Error = err.Error()
			report.FailureCount++
			o.logger.ErrorContext(ctx, "push failed",
				"provider", target.ProviderName, "error", err)
		} else {
			op.Status = model.StatusCompleted
			report.SuccessCount++
			o.logger.InfoContext(ctx, "push succeeded",
				"provider", target.ProviderName)
		}

		op.UpdatedAt = time.Now()
		report.Operations = append(report.Operations, op)
	}

	report.EndTime = time.Now()
	report.DurationSeconds = report.EndTime.Sub(report.StartTime).Seconds()

	return report, nil
}

// Status checks the sync state of all providers for a repo.
func (o *Orchestrator) Status(
	ctx context.Context,
	repoPath string,
	providers map[string]provider.Provider,
	targets []model.RepoSyncTarget,
) (*model.StatusReport, error) {
	o.logger.DebugContext(ctx, "checking status", "repo", repoPath)

	localHead, err := o.branchMgr.GetHeadHash(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("get local head: %w", err)
	}

	currentBranch, err := o.branchMgr.CurrentBranch(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	report := &model.StatusReport{
		RepoPath:    repoPath,
		LocalHead:   localHead,
		LocalBranch: currentBranch,
	}

	for _, target := range targets {
		if !target.Enabled {
			continue
		}

		prov, ok := providers[target.ProviderName]
		if !ok {
			continue
		}

		ps := model.ProviderStatus{
			Name: target.ProviderName,
			Type: string(prov.Type()),
		}

		remoteName := prov.RemoteName()
		remoteHead, err := o.remoteMgr.GetRemoteHeadHash(ctx, repoPath, remoteName, currentBranch)
		if err != nil {
			ps.Error = err.Error()
			ps.InSync = false
		} else {
			ps.RemoteHead = remoteHead
			ps.InSync = remoteHead == localHead
		}

		report.Providers = append(report.Providers, ps)
	}

	return report, nil
}

// Fetch synchronizes remote changes from all specified providers to local.
func (o *Orchestrator) Fetch(
	ctx context.Context,
	repoPath string,
	providers map[string]provider.Provider,
	targets []model.RepoSyncTarget,
) (*SyncReport, error) {
	o.logger.InfoContext(ctx, "starting fetch", "repo", repoPath)

	report := &SyncReport{
		OperationType: model.OpFetch,
		RepoPath:      repoPath,
		TotalTargets:  len(targets),
		StartTime:     time.Now(),
		Operations:    []model.SyncOperation{},
	}

	for _, target := range targets {
		if !target.Enabled {
			continue
		}

		prov, ok := providers[target.ProviderName]
		if !ok {
			continue
		}

		remoteName := prov.RemoteName()
		op := model.SyncOperation{
			ID:           fmt.Sprintf("%s-%s-fetch-%d", repoPath, target.ProviderName, time.Now().UnixNano()),
			ProviderName: target.ProviderName,
			RepoPath:     repoPath,
			Type:         model.OpFetch,
			Status:       model.StatusInProgress,
			CreatedAt:    time.Now(),
		}

		if err := o.remoteMgr.Fetch(ctx, repoPath, remoteName); err != nil {
			op.Status = model.StatusFailed
			op.Error = err.Error()
			report.FailureCount++
		} else {
			op.Status = model.StatusCompleted
			report.SuccessCount++
		}

		op.UpdatedAt = time.Now()
		report.Operations = append(report.Operations, op)
	}

	report.EndTime = time.Now()
	report.DurationSeconds = report.EndTime.Sub(report.StartTime).Seconds()

	return report, nil
}

// executePush performs a single git push to a provider's remote.
// It now also handles repository creation if needed.
func (o *Orchestrator) executePush(
	ctx context.Context,
	repoPath string,
	prov provider.Provider,
	target model.RepoSyncTarget,
	branch string,
) error {
	remoteName := prov.RemoteName()

	// 1. Ensure remote exists in git config
	if !o.remoteMgr.HasRemote(ctx, repoPath, remoteName) {
		if err := o.remoteMgr.AddRemote(ctx, repoPath, remoteName, target.RemoteURL); err != nil {
			return fmt.Errorf("add remote %s: %w", remoteName, err)
		}
	}

	// 2. Check if repository exists and create if needed
	// Note: This is optional - repositories may already exist
	// Only attempt if we have credentials for metadata operations
	if o.credMgr != nil {
		if err := o.ensureRepositoryExists(ctx, prov, target); err != nil {
			o.logger.WarnContext(ctx, "failed to ensure repository exists",
				"provider", target.ProviderName, "error", err)
			// Don't fail the push if repo creation fails - just warn
		}
	}

	// 3. Try to inject credentials into git before pushing
	// This pre-populates git's credential cache so it doesn't prompt
	if o.credMgr != nil && prov != nil {
		// Build provider config for credential resolution
		providerCfg := model.ProviderConfig{
			Name: target.ProviderName,
		}

		// Resolve credentials for the provider
		cred, err := o.credMgr.Resolve(ctx, providerCfg)
		if err == nil && cred != nil && cred.Token != "" {
			// Pre-store credentials in git's cache using `git credential approve`
			if err := o.remoteMgr.PushWithCredential(ctx, repoPath, remoteName, target.RemoteURL,
				[]string{branch}, true, false, "git", cred.Token); err != nil {
				return fmt.Errorf("push to %s: %w", remoteName, err)
			}
		} else {
			// Fall back to regular push without credential injection
			if err := o.remoteMgr.Push(ctx, repoPath, remoteName, []string{branch}, true, false); err != nil {
				return fmt.Errorf("push to %s: %w", remoteName, err)
			}
		}
	} else {
		// No credential manager, use regular push
		if err := o.remoteMgr.Push(ctx, repoPath, remoteName, []string{branch}, true, false); err != nil {
			return fmt.Errorf("push to %s: %w", remoteName, err)
		}
	}

	return nil
}

// ensureRepositoryExists creates a repository if it doesn't already exist.
// It uses default metadata if no specific metadata is provided.
func (o *Orchestrator) ensureRepositoryExists(
	ctx context.Context,
	prov provider.Provider,
	target model.RepoSyncTarget,
) error {
	// Note: We need provider config to resolve credentials, but we only have provider name
	// For now, we'll skip credential resolution and return early
	// In a full implementation, we'd pass provider config to this function
	o.logger.DebugContext(ctx, "skipping repo creation (credential resolution not yet integrated)",
		"provider", target.ProviderName)
	return nil

}
