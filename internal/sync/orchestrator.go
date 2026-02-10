package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
	logger    *slog.Logger
}

// NewOrchestrator creates a new sync orchestrator.
func NewOrchestrator(gitExec git.GitExecutor, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{
		gitExec:   gitExec,
		remoteMgr: git.NewRemoteManager(gitExec, logger),
		branchMgr: git.NewBranchManager(gitExec, logger),
		logger:    logger,
	}
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

	// 2. Push current branch
	if err := o.remoteMgr.Push(ctx, repoPath, remoteName, []string{branch}, true, false); err != nil {
		return fmt.Errorf("push to %s: %w", remoteName, err)
	}

	return nil
}
