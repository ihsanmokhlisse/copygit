package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/sync"
)

// Format specifies the output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Formatter handles output formatting for all data structures.
// Per contracts/internal-interfaces.md, the Formatter interface
// has up to 8 methods.
type Formatter interface {
	PrintSyncReport(report *sync.SyncReport) error
	PrintStatusReport(report *model.StatusReport) error
	PrintRepoList(repos []model.RepoRegistration) error
	PrintMultiRepoSyncReport(reports []*sync.SyncReport) error
	PrintProviderList(providers []model.ProviderConfig) error
	PrintError(err error)
	PrintSuccess(msg string)
	PrintWarning(msg string)
}

// ---- TextFormatter ----

// TextFormatter outputs human-readable text.
type TextFormatter struct {
	out io.Writer
}

// NewTextFormatter creates a text formatter.
func NewTextFormatter(out io.Writer) *TextFormatter {
	return &TextFormatter{out: out}
}

func (t *TextFormatter) PrintSyncReport(report *sync.SyncReport) error {
	fmt.Fprintf(t.out, "Sync Report: %s\n", report.OperationType)
	fmt.Fprintf(t.out, "Repository: %s\n", report.RepoPath)
	fmt.Fprintf(t.out, "Duration: %.2f seconds\n", report.DurationSeconds)
	fmt.Fprintf(t.out, "Targets: %d (%d success, %d failures)\n",
		report.TotalTargets, report.SuccessCount, report.FailureCount)

	if len(report.ReposCreated) > 0 {
		fmt.Fprintf(t.out, "\nRepositories Created:\n")
		for _, repo := range report.ReposCreated {
			fmt.Fprintf(t.out, "  ✓ %s\n", repo)
		}
	}

	if len(report.MetadataSynced) > 0 {
		fmt.Fprintf(t.out, "\nMetadata Synced:\n")
		for _, meta := range report.MetadataSynced {
			fmt.Fprintf(t.out, "  ✓ %s\n", meta)
		}
	}

	if len(report.MetadataWarnings) > 0 {
		fmt.Fprintf(t.out, "\nMetadata Warnings:\n")
		for _, warn := range report.MetadataWarnings {
			fmt.Fprintf(t.out, "  ⚠ %s\n", warn)
		}
	}

	if len(report.Operations) > 0 {
		fmt.Fprintf(t.out, "\nOperation Details:\n")
		for i := range report.Operations {
			op := report.Operations[i]
			status := string(op.Status)
			if op.Error != "" {
				fmt.Fprintf(t.out, "  [%s] %s: %s\n", status, op.ProviderName, op.Error)
			} else {
				fmt.Fprintf(t.out, "  [%s] %s\n", status, op.ProviderName)
			}
		}
	}
	return nil
}

func (t *TextFormatter) PrintStatusReport(report *model.StatusReport) error {
	fmt.Fprintf(t.out, "Status: %s\n", report.RepoPath)
	headShort := report.LocalHead
	if len(headShort) > 8 {
		headShort = headShort[:8]
	}
	fmt.Fprintf(t.out, "Branch: %s (%s)\n", report.LocalBranch, headShort)
	fmt.Fprintf(t.out, "Queued operations: %d\n\n", report.QueuedOps)

	w := tabwriter.NewWriter(t.out, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "PROVIDER\tTYPE\tIN SYNC\tREMOTE HEAD\tLAST SYNC\n")
	for _, p := range report.Providers {
		inSync := "no"
		if p.InSync {
			inSync = "yes"
		}
		remoteHead := p.RemoteHead
		if len(remoteHead) > 8 {
			remoteHead = remoteHead[:8]
		}
		lastSync := p.LastSyncTime
		if lastSync == "" {
			lastSync = "never"
		}
		errStr := ""
		if p.Error != "" {
			errStr = " (" + p.Error + ")"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s%s\n",
			p.Name, p.Type, inSync, remoteHead, lastSync, errStr)
	}
	w.Flush()
	return nil
}

func (t *TextFormatter) PrintRepoList(repos []model.RepoRegistration) error {
	if len(repos) == 0 {
		fmt.Fprintf(t.out, "No repositories registered.\n")
		return nil
	}

	w := tabwriter.NewWriter(t.out, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "PATH\tALIAS\tLAST SYNC\n")
	for _, repo := range repos {
		alias := repo.Alias
		if alias == "" {
			alias = "-"
		}
		lastSync := "never"
		if !repo.LastSyncTime.IsZero() {
			lastSync = repo.LastSyncTime.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", repo.Path, alias, lastSync)
	}
	w.Flush()
	return nil
}

func (t *TextFormatter) PrintMultiRepoSyncReport(reports []*sync.SyncReport) error {
	totalSuccess := 0
	totalFailures := 0
	for _, r := range reports {
		totalSuccess += r.SuccessCount
		totalFailures += r.FailureCount
	}

	fmt.Fprintf(t.out, "\nMulti-Repo Sync Summary\n")
	fmt.Fprintf(t.out, "Repositories: %d\n", len(reports))
	fmt.Fprintf(t.out, "Total: %d success, %d failures\n", totalSuccess, totalFailures)
	return nil
}

func (t *TextFormatter) PrintProviderList(providers []model.ProviderConfig) error {
	if len(providers) == 0 {
		fmt.Fprintf(t.out, "No providers configured.\n")
		return nil
	}

	w := tabwriter.NewWriter(t.out, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tTYPE\tBASE URL\tAUTH\tPREFERRED\n")
	for _, prov := range providers {
		preferred := ""
		if prov.IsPreferred {
			preferred = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			prov.Name, prov.Type, prov.BaseURL, prov.AuthMethod, preferred)
	}
	w.Flush()
	return nil
}

func (t *TextFormatter) PrintError(err error) {
	fmt.Fprintf(t.out, "Error: %v\n", err)
}

func (t *TextFormatter) PrintSuccess(msg string) {
	fmt.Fprintf(t.out, "%s\n", msg)
}

func (t *TextFormatter) PrintWarning(msg string) {
	fmt.Fprintf(t.out, "Warning: %s\n", msg)
}

// ---- JSONFormatter ----

// JSONFormatter outputs machine-readable JSON.
type JSONFormatter struct {
	out io.Writer
}

// NewJSONFormatter creates a JSON formatter.
func NewJSONFormatter(out io.Writer) *JSONFormatter {
	return &JSONFormatter{out: out}
}

func (j *JSONFormatter) PrintSyncReport(report *sync.SyncReport) error {
	return j.writeJSON(report)
}

func (j *JSONFormatter) PrintStatusReport(report *model.StatusReport) error {
	return j.writeJSON(report)
}

func (j *JSONFormatter) PrintRepoList(repos []model.RepoRegistration) error {
	return j.writeJSON(repos)
}

func (j *JSONFormatter) PrintMultiRepoSyncReport(reports []*sync.SyncReport) error {
	return j.writeJSON(reports)
}

func (j *JSONFormatter) PrintProviderList(providers []model.ProviderConfig) error {
	return j.writeJSON(providers)
}

func (j *JSONFormatter) PrintError(err error) {
	_ = j.writeJSON(map[string]string{"error": err.Error()})
}

func (j *JSONFormatter) PrintSuccess(msg string) {
	_ = j.writeJSON(map[string]string{"message": msg})
}

func (j *JSONFormatter) PrintWarning(msg string) {
	_ = j.writeJSON(map[string]string{"warning": msg})
}

func (j *JSONFormatter) writeJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(j.out, "%s\n", data)
	return nil
}

// ---- FakeFormatter ----

// FakeFormatter captures calls for testing.
type FakeFormatter struct {
	SyncReports   []*sync.SyncReport
	StatusReports []*model.StatusReport
	Errors        []error
	Successes     []string
	Warnings      []string
}

func (f *FakeFormatter) PrintSyncReport(r *sync.SyncReport) error {
	f.SyncReports = append(f.SyncReports, r)
	return nil
}

func (f *FakeFormatter) PrintStatusReport(r *model.StatusReport) error {
	f.StatusReports = append(f.StatusReports, r)
	return nil
}

func (f *FakeFormatter) PrintRepoList(_ []model.RepoRegistration) error { return nil }

func (f *FakeFormatter) PrintMultiRepoSyncReport(_ []*sync.SyncReport) error { return nil }

func (f *FakeFormatter) PrintProviderList(_ []model.ProviderConfig) error { return nil }

func (f *FakeFormatter) PrintError(err error) { f.Errors = append(f.Errors, err) }

func (f *FakeFormatter) PrintSuccess(msg string) { f.Successes = append(f.Successes, msg) }

func (f *FakeFormatter) PrintWarning(msg string) { f.Warnings = append(f.Warnings, msg) }
