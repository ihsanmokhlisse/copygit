package daemon

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/provider"
	"github.com/imokhlis/copygit/internal/sync"
)

// Daemon monitors and periodically syncs registered repositories.
type Daemon struct {
	pollInterval time.Duration
	logger       *slog.Logger
	gitExec      git.GitExecutor
	credMgr      credential.Manager
	globalCfg    *config.GlobalConfig
	registry     *model.RepoRegistry
	registryPath string
}

// NewDaemon creates a new daemon instance.
func NewDaemon(
	pollInterval time.Duration,
	logger *slog.Logger,
	gitExec git.GitExecutor,
	credMgr credential.Manager,
	globalCfg *config.GlobalConfig,
	registry *model.RepoRegistry,
	registryPath string,
) *Daemon {
	return &Daemon{
		pollInterval: pollInterval,
		logger:       logger,
		gitExec:      gitExec,
		credMgr:      credMgr,
		globalCfg:    globalCfg,
		registry:     registry,
		registryPath: registryPath,
	}
}

// Run starts the daemon (blocking until context is cancelled).
func (d *Daemon) Run(ctx context.Context) error {
	d.logger.InfoContext(ctx, "daemon started", "poll_interval", d.pollInterval)

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.InfoContext(ctx, "daemon stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := d.syncAllRepos(ctx); err != nil {
				d.logger.ErrorContext(ctx, "sync cycle error", "error", err)
			}
		}
	}
}

// syncAllRepos syncs all registered repositories once.
func (d *Daemon) syncAllRepos(ctx context.Context) error { //nolint:unparam // error return reserved for future use
	d.logger.DebugContext(ctx, "polling for syncs")

	if len(d.registry.Repos) == 0 {
		return nil
	}

	orchestrator := sync.NewOrchestrator(d.gitExec, d.logger).WithCredentialManager(d.credMgr)

	for _, repo := range d.registry.Repos {
		repoCfg, err := config.LoadRepoConfig(repo.Path)
		if err != nil {
			d.logger.WarnContext(ctx, "load repo config", "repo", repo.Path, "error", err)
			continue
		}

		enabledNames := repoCfg.EnabledProviderNames()
		providerConfigs, err := d.globalCfg.ProvidersByNames(enabledNames)
		if err != nil {
			d.logger.WarnContext(ctx, "resolve providers", "repo", repo.Path, "error", err)
			continue
		}

		providerRegistry, err := provider.BuildRegistry(ctx, providerConfigs, d.credMgr, d.gitExec, d.logger)
		if err != nil {
			d.logger.WarnContext(ctx, "build registry", "repo", repo.Path, "error", err)
			continue
		}

		providerMap := make(map[string]provider.Provider)
		for _, name := range enabledNames {
			if prov, err := providerRegistry.Get(name); err == nil {
				providerMap[name] = prov
			}
		}

		_, err = orchestrator.Push(ctx, repo.Path, providerMap, repoCfg.AsRepoSyncTargets())
		if err != nil {
			d.logger.WarnContext(ctx, "sync failed", "repo", repo.Path, "error", err)
		} else {
			d.logger.InfoContext(ctx, "sync completed", "repo", repo.Path)
			_ = config.UpdateLastSync(d.registry, repo.Path)
		}
	}

	return nil
}

// SyncQueue holds failed sync operations for retry.
type SyncQueue struct {
	queueDir string
	logger   *slog.Logger
}

// NewSyncQueue creates a new sync queue.
func NewSyncQueue(queueDir string, logger *slog.Logger) *SyncQueue {
	return &SyncQueue{queueDir: queueDir, logger: logger}
}

// Enqueue persists a failed sync operation for later retry.
func (sq *SyncQueue) Enqueue(ctx context.Context, op *OpQueueItem) error {
	sq.logger.DebugContext(ctx, "enqueue operation", "id", op.ID)
	// TODO: Write to queueDir/op.ID.json
	return nil
}

// Dequeue retrieves a failed operation for retry.
func (sq *SyncQueue) Dequeue(ctx context.Context, id string) (*OpQueueItem, error) { //nolint:revive // required by SyncQueue interface
	// TODO: Read from queueDir/id.json
	return nil, errors.New("not implemented")
}

// OpQueueItem represents a failed sync operation to retry.
type OpQueueItem struct {
	ID         string
	Operation  *model.SyncOperation
	RetryCount int
	MaxRetries int
	LastError  string
}
