#!/usr/bin/env bash
# test-orchestrator.sh — Intelligent test orchestration for CopyGit
#
# Usage:
#   ./scripts/test-orchestrator.sh              # Run full pipeline (smart order)
#   ./scripts/test-orchestrator.sh fast          # Fast unit tests only (~2s)
#   ./scripts/test-orchestrator.sh changed       # Only packages with changed files
#   ./scripts/test-orchestrator.sh coverage      # Full coverage with threshold check
#   ./scripts/test-orchestrator.sh ci            # Full CI pipeline (lint → test → race → coverage)
#
# Environment variables:
#   COVERAGE_THRESHOLD  Minimum coverage % to pass (default: 35)
#   VERBOSE             Set to 1 for verbose test output
#   FAIL_FAST           Set to 1 to stop on first failure (default: 0)

set -euo pipefail

# ─── Configuration ────────────────────────────────────────────────────────────

COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-35}"
VERBOSE="${VERBOSE:-0}"
FAIL_FAST="${FAIL_FAST:-0}"
COVERFILE="cover.out"

# Color codes (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; BOLD=''; NC=''
fi

# ─── Package classification ──────────────────────────────────────────────────
#
# Tier 1 (Pure logic, no I/O, ~0.5s):
#   model, git (fakes only), output
#
# Tier 2 (File I/O, temp dirs, ~1s):
#   config, credential, daemon, hook, lock
#
# Tier 3 (External deps, network, concurrency, ~2s):
#   provider (httptest servers), sync (orchestration)
#
# Tier 4 (Integration tests, real git repos):
#   config/integration, hook/integration

TIER1="./internal/model/... ./internal/git/... ./internal/output/..."
TIER2="./internal/config/... ./internal/credential/... ./internal/daemon/... ./internal/lock/..."
TIER3="./internal/provider/... ./internal/sync/... ./internal/hook/..."

# Concurrency-sensitive packages that need race detection
RACE_PACKAGES="./internal/lock/... ./internal/sync/... ./internal/daemon/..."

# ─── Helpers ──────────────────────────────────────────────────────────────────

log_step() { echo -e "${BLUE}${BOLD}==>${NC} $1"; }
log_pass() { echo -e "${GREEN}  PASS${NC} $1"; }
log_fail() { echo -e "${RED}  FAIL${NC} $1"; }
log_warn() { echo -e "${YELLOW}  WARN${NC} $1"; }
log_info() { echo -e "  $1"; }

timer_start() { TIMER_START=$(date +%s%N); }
timer_end() {
    local elapsed=$(( ($(date +%s%N) - TIMER_START) / 1000000 ))
    if [ "$elapsed" -gt 1000 ]; then
        echo "$((elapsed / 1000)).$((elapsed % 1000 / 100))s"
    else
        echo "${elapsed}ms"
    fi
}

test_flags() {
    local flags="-count=1"
    [ "$VERBOSE" = "1" ] && flags="$flags -v"
    [ "$FAIL_FAST" = "1" ] && flags="$flags -failfast"
    echo "$flags"
}

run_tests() {
    local label="$1"; shift
    local packages="$*"

    timer_start
    if go test $(test_flags) $packages 2>&1; then
        log_pass "$label ($(timer_end))"
        return 0
    else
        log_fail "$label ($(timer_end))"
        return 1
    fi
}

# ─── Commands ─────────────────────────────────────────────────────────────────

cmd_fast() {
    log_step "Fast unit tests (Tier 1 — pure logic)"
    run_tests "Tier 1: model, git, output" $TIER1
}

cmd_changed() {
    log_step "Testing changed packages only"

    # Find Go files changed since last commit
    local changed_files
    changed_files=$(git diff --name-only HEAD -- '*.go' 2>/dev/null || true)
    changed_files="$changed_files $(git diff --cached --name-only -- '*.go' 2>/dev/null || true)"

    if [ -z "$changed_files" ]; then
        log_info "No changed Go files detected"
        return 0
    fi

    # Extract unique package directories
    local packages=""
    for f in $changed_files; do
        local dir
        dir=$(dirname "$f")
        if [ -n "$dir" ] && [ "$dir" != "." ]; then
            packages="$packages ./$dir/..."
        fi
    done

    # Deduplicate
    packages=$(echo "$packages" | tr ' ' '\n' | sort -u | tr '\n' ' ')

    if [ -z "$packages" ]; then
        log_info "No testable packages changed"
        return 0
    fi

    log_info "Changed packages: $packages"
    run_tests "Changed packages" $packages
}

cmd_coverage() {
    log_step "Coverage analysis (threshold: ${COVERAGE_THRESHOLD}%)"

    timer_start
    go test -coverprofile="$COVERFILE" -covermode=atomic ./... > /dev/null 2>&1

    # Extract total coverage
    local total
    total=$(go tool cover -func="$COVERFILE" 2>/dev/null | tail -1 | awk '{print $NF}' | tr -d '%')

    # Per-package coverage
    echo ""
    log_info "Per-package coverage:"
    go test -coverprofile=/dev/null ./... 2>&1 | grep 'coverage:' | while read -r line; do
        local pkg pct
        pkg=$(echo "$line" | awk '{print $2}')
        pct=$(echo "$line" | grep -oE '[0-9]+\.[0-9]+%')

        # Short package name
        pkg="${pkg#github.com/imokhlis/copygit/}"

        local pct_num="${pct%\%}"
        if (( $(echo "$pct_num >= 80" | bc -l) )); then
            log_pass "$pkg: $pct"
        elif (( $(echo "$pct_num >= 50" | bc -l) )); then
            log_warn "$pkg: $pct"
        else
            log_fail "$pkg: $pct"
        fi
    done

    echo ""

    # Threshold check
    if (( $(echo "$total >= $COVERAGE_THRESHOLD" | bc -l) )); then
        log_pass "Total coverage: ${total}% >= ${COVERAGE_THRESHOLD}% threshold ($(timer_end))"
    else
        log_fail "Total coverage: ${total}% < ${COVERAGE_THRESHOLD}% threshold ($(timer_end))"
        return 1
    fi

    # Generate HTML report
    go tool cover -html="$COVERFILE" -o cover.html 2>/dev/null
    log_info "HTML report: cover.html"
}

cmd_race() {
    log_step "Race detector (concurrency-sensitive packages)"
    run_tests "Race detection" -race $RACE_PACKAGES
}

cmd_lint() {
    log_step "Static analysis"

    timer_start
    local lint_cmd="golangci-lint"
    if ! command -v "$lint_cmd" &> /dev/null; then
        local gopath_lint
        gopath_lint="$(go env GOPATH)/bin/golangci-lint"
        if [ -x "$gopath_lint" ]; then
            lint_cmd="$gopath_lint"
        fi
    fi
    if ! command -v "$lint_cmd" &> /dev/null && [ ! -x "$lint_cmd" ]; then
        log_warn "golangci-lint not installed, running go vet only"
        if go vet ./... 2>&1; then
            log_pass "go vet ($(timer_end))"
        else
            log_fail "go vet ($(timer_end))"
            return 1
        fi
        return 0
    fi

    if $lint_cmd run 2>&1; then
        log_pass "golangci-lint ($(timer_end))"
    else
        log_fail "golangci-lint ($(timer_end))"
        return 1
    fi
}

cmd_build() {
    log_step "Build verification"

    timer_start
    if go build -o /dev/null ./cmd/copygit 2>&1; then
        log_pass "Build ($(timer_end))"
    else
        log_fail "Build ($(timer_end))"
        return 1
    fi
}

cmd_full() {
    log_step "Full test pipeline"
    echo ""

    local failed=0

    # Phase 1: Fast feedback (< 1s)
    run_tests "Tier 1: Pure logic (model, git, output)" $TIER1 || failed=1
    [ "$FAIL_FAST" = "1" ] && [ "$failed" -ne 0 ] && return 1

    # Phase 2: File I/O tests (< 2s)
    run_tests "Tier 2: File I/O (config, credential, daemon, lock)" $TIER2 || failed=1
    [ "$FAIL_FAST" = "1" ] && [ "$failed" -ne 0 ] && return 1

    # Phase 3: External deps (< 3s)
    run_tests "Tier 3: External (provider, sync, hook)" $TIER3 || failed=1
    [ "$FAIL_FAST" = "1" ] && [ "$failed" -ne 0 ] && return 1

    echo ""
    if [ "$failed" -eq 0 ]; then
        log_pass "All tiers passed"
    else
        log_fail "Some tiers failed"
    fi

    return $failed
}

cmd_ci() {
    log_step "CI Pipeline — full verification"
    echo ""

    local start_time
    start_time=$(date +%s)
    local failed=0

    # Stage 1: Build
    cmd_build || { failed=1; [ "$FAIL_FAST" = "1" ] && return 1; }

    # Stage 2: Lint
    cmd_lint || { failed=1; [ "$FAIL_FAST" = "1" ] && return 1; }

    # Stage 3: Tests (tiered)
    cmd_full || { failed=1; [ "$FAIL_FAST" = "1" ] && return 1; }

    # Stage 4: Race detection
    cmd_race || { failed=1; [ "$FAIL_FAST" = "1" ] && return 1; }

    # Stage 5: Coverage threshold
    cmd_coverage || { failed=1; [ "$FAIL_FAST" = "1" ] && return 1; }

    echo ""
    local elapsed=$(( $(date +%s) - start_time ))
    if [ "$failed" -eq 0 ]; then
        log_pass "CI pipeline passed (${elapsed}s total)"
    else
        log_fail "CI pipeline failed (${elapsed}s total)"
    fi

    return $failed
}

# ─── Main ─────────────────────────────────────────────────────────────────────

main() {
    local command="${1:-full}"

    echo -e "${BOLD}CopyGit Test Orchestrator${NC}"
    echo ""

    case "$command" in
        fast)     cmd_fast ;;
        changed)  cmd_changed ;;
        coverage) cmd_coverage ;;
        race)     cmd_race ;;
        lint)     cmd_lint ;;
        build)    cmd_build ;;
        full)     cmd_full ;;
        ci)       cmd_ci ;;
        *)
            echo "Usage: $0 {fast|changed|coverage|race|lint|build|full|ci}"
            echo ""
            echo "Commands:"
            echo "  fast      Tier 1 only (pure logic, ~0.5s)"
            echo "  changed   Only packages with uncommitted changes"
            echo "  coverage  Coverage report with threshold check"
            echo "  race      Race detector on concurrency-sensitive packages"
            echo "  lint      Static analysis (golangci-lint or go vet)"
            echo "  build     Build verification"
            echo "  full      All test tiers (default)"
            echo "  ci        Full CI pipeline (build → lint → test → race → coverage)"
            exit 1
            ;;
    esac
}

main "$@"
