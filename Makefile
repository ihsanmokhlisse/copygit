.PHONY: build test test-fast test-changed test-race test-integration coverage lint fmt clean install ci

# ─── Build ────────────────────────────────────────────────────────────────────

# Build the binary
build:
	go build -o bin/copygit ./cmd/copygit

# Install to GOPATH/bin
install:
	go install ./cmd/copygit

# ─── Testing (tiered) ────────────────────────────────────────────────────────

# Run all unit tests:
test:
	go test -count=1 ./...

# Tier 1 only — pure logic, no I/O (~0.5s):
test-fast:
	go test -count=1 ./internal/model/... ./internal/git/... ./internal/output/...

# Only packages with uncommitted changes:
test-changed:
	@./scripts/test-orchestrator.sh changed

# Run tests with race detector (concurrency-sensitive packages):
test-race:
	go test -race -count=1 ./internal/lock/... ./internal/sync/... ./internal/daemon/...

# Run integration tests (creates temp git repos on disk):
test-integration:
	go test -v -tags=integration ./...

# ─── Coverage ─────────────────────────────────────────────────────────────────

# Generate coverage report with threshold check:
coverage:
	@./scripts/test-orchestrator.sh coverage

# Coverage report (simple, no threshold):
coverage-html:
	go test -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html
	@echo "Coverage report: cover.html"

# ─── Code quality ────────────────────────────────────────────────────────────

# Run linter:
lint:
	golangci-lint run

# Format code:
fmt:
	goimports -w .
	golines -w --max-len=120 .

# Run go vet:
vet:
	go vet ./...

# ─── CI pipeline ─────────────────────────────────────────────────────────────

# Full CI pipeline (build → lint → test → race → coverage):
ci:
	@./scripts/test-orchestrator.sh ci

# Quick pre-commit check (build + fast tests + vet):
precommit: build vet test-fast

# ─── Cleanup ─────────────────────────────────────────────────────────────────

clean:
	rm -rf bin/ cover.out cover.html
