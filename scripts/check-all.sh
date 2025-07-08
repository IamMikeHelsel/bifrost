#!/bin/bash
# check-all.sh - Comprehensive quality check for Bifrost project
# Runs linters, formatters, tests, and provides issue summary

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Counters for issues
TOTAL_ERRORS=0
TOTAL_WARNINGS=0
TOTAL_TESTS=0
FAILED_TESTS=0

# Helper functions
log_section() {
    echo -e "\n${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

log_step() {
    echo -e "\n${CYAN}→ $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

count_issues() {
    local output="$1"
    local errors=$(echo "$output" | grep -c "Error\|ERROR\|✘" || true)
    local warnings=$(echo "$output" | grep -c "Warning\|WARN\|⚠" || true)
    TOTAL_ERRORS=$((TOTAL_ERRORS + errors))
    TOTAL_WARNINGS=$((TOTAL_WARNINGS + warnings))
    echo "Errors: $errors, Warnings: $warnings"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Start timing
START_TIME=$(date +%s)

echo -e "${PURPLE}🌉 Bifrost Quality Check Suite${NC}"
echo -e "${PURPLE}==============================${NC}"
echo "Starting comprehensive code quality analysis..."

# =============================================================================
# 1. CODE FORMATTING
# =============================================================================
log_section "📝 CODE FORMATTING"

# Go formatting
if command_exists go; then
    log_step "Formatting Go code"
    if go fmt ./...; then
        log_success "Go code formatted"
    else
        log_error "Go formatting failed"
        ((TOTAL_ERRORS++))
    fi
else
    log_warning "Go not found, skipping Go formatting"
fi

# Python formatting with ruff
if command_exists ruff; then
    log_step "Formatting Python code with ruff"
    output=$(ruff format packages/ 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Python code formatted with ruff"
    else
        log_error "Python formatting failed"
        echo "$output"
        ((TOTAL_ERRORS++))
    fi
else
    log_warning "ruff not found, skipping Python formatting"
fi

# TypeScript/JavaScript formatting (if VS Code extension exists)
if [ -d "vscode-extension" ] && command_exists npm; then
    log_step "Formatting TypeScript code"
    cd vscode-extension
    if npm run format --silent 2>/dev/null; then
        log_success "TypeScript code formatted"
    else
        log_warning "TypeScript formatting failed or not configured"
    fi
    cd ..
fi

# Markdown formatting
if command_exists markdownlint; then
    log_step "Checking Markdown formatting"
    output=$(markdownlint README.md docs/ 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Markdown formatting OK"
    else
        log_warning "Markdown formatting issues found"
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
else
    log_warning "markdownlint not found, skipping Markdown checks"
fi

# =============================================================================
# 2. CODE LINTING
# =============================================================================
log_section "🔍 CODE LINTING"

# Go linting
if command_exists golangci-lint; then
    log_step "Linting Go code with golangci-lint"
    output=$(golangci-lint run ./... 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Go linting passed"
    else
        log_error "Go linting issues found"
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
elif command_exists go; then
    log_step "Running go vet (basic Go linting)"
    output=$(go vet ./... 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Go vet passed"
    else
        log_error "Go vet issues found"
        echo "$output"
        ((TOTAL_ERRORS++))
    fi
else
    log_warning "No Go linting tools found"
fi

# Python linting with ruff
if command_exists ruff; then
    log_step "Linting Python code with ruff"
    output=$(ruff check packages/ 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Python linting passed"
    else
        log_warning "Python linting issues found"
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
else
    log_warning "ruff not found, skipping Python linting"
fi

# Python type checking with mypy
if command_exists mypy; then
    log_step "Type checking Python code with mypy"
    output=$(mypy packages/bifrost/src/ packages/bifrost-core/src/ 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Python type checking passed"
    else
        log_warning "Python type checking issues found"
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
else
    log_warning "mypy not found, skipping Python type checking"
fi

# TypeScript linting (if VS Code extension exists)
if [ -d "vscode-extension" ] && command_exists npm; then
    log_step "Linting TypeScript code"
    cd vscode-extension
    if npm run lint --silent 2>/dev/null; then
        log_success "TypeScript linting passed"
    else
        log_warning "TypeScript linting issues found"
        output=$(npm run lint 2>&1 || true)
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
    cd ..
fi

# =============================================================================
# 3. SECURITY SCANNING
# =============================================================================
log_section "🔒 SECURITY SCANNING"

# Go security scanning
if command_exists gosec; then
    log_step "Scanning Go code for security issues"
    output=$(gosec ./... 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Go security scan passed"
    else
        log_warning "Go security issues found"
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
else
    log_warning "gosec not found, skipping Go security scan"
fi

# Go vulnerability check
if command_exists govulncheck; then
    log_step "Checking Go dependencies for vulnerabilities"
    output=$(govulncheck ./... 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Go vulnerability check passed"
    else
        log_warning "Go vulnerabilities found"
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
else
    log_warning "govulncheck not found, skipping Go vulnerability check"
fi

# Python security scanning
if command_exists bandit; then
    log_step "Scanning Python code for security issues"
    output=$(bandit -r packages/ 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Python security scan passed"
    else
        log_warning "Python security issues found"
        echo "$output"
        issues=$(count_issues "$output")
        echo "Issues: $issues"
    fi
else
    log_warning "bandit not found, skipping Python security scan"
fi

# =============================================================================
# 4. TESTING
# =============================================================================
log_section "🧪 TESTING"

# Go tests
if command_exists go; then
    log_step "Running Go tests (with 60s timeout)"
    output=$(timeout 60s go test -v ./... 2>&1 || true)
    test_result=$?
    
    # Count tests
    go_tests=$(echo "$output" | grep -c "RUN\|PASS\|FAIL" || true)
    go_failures=$(echo "$output" | grep -c "FAIL:" || true)
    
    TOTAL_TESTS=$((TOTAL_TESTS + go_tests))
    FAILED_TESTS=$((FAILED_TESTS + go_failures))
    
    if [[ $test_result -eq 0 ]]; then
        log_success "Go tests passed ($go_tests tests)"
    else
        log_error "Go tests failed ($go_failures failures out of $go_tests tests)"
        echo "$output"
    fi

    # Go tests with race detection
    log_step "Running Go tests with race detection"
    race_output=$(go test -race ./... 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Go race tests passed"
    else
        log_warning "Go race tests found issues"
        echo "$race_output"
        ((TOTAL_WARNINGS++))
    fi
else
    log_warning "Go not found, skipping Go tests"
fi

# Python tests
if command_exists pytest; then
    log_step "Running Python tests (with 120s timeout)"
    output=$(timeout 120s pytest packages/bifrost/tests/ packages/bifrost-core/tests/ -v 2>&1 || true)
    test_result=$?
    
    # Count tests
    python_tests=$(echo "$output" | grep -c "PASSED\|FAILED\|ERROR" || true)
    python_failures=$(echo "$output" | grep -c "FAILED\|ERROR" || true)
    
    TOTAL_TESTS=$((TOTAL_TESTS + python_tests))
    FAILED_TESTS=$((FAILED_TESTS + python_failures))
    
    if [[ $test_result -eq 0 ]]; then
        log_success "Python tests passed ($python_tests tests)"
    else
        log_error "Python tests failed ($python_failures failures out of $python_tests tests)"
        echo "$output"
    fi
else
    log_warning "pytest not found, skipping Python tests"
fi

# TypeScript tests (if VS Code extension exists)
if [ -d "vscode-extension" ] && command_exists npm; then
    log_step "Running TypeScript tests"
    cd vscode-extension
    if npm test --silent 2>/dev/null; then
        log_success "TypeScript tests passed"
    else
        log_warning "TypeScript tests failed or not configured"
        output=$(npm test 2>&1 || true)
        echo "$output"
        ts_failures=$(echo "$output" | grep -c "failed\|error" || true)
        FAILED_TESTS=$((FAILED_TESTS + ts_failures))
    fi
    cd ..
fi

# =============================================================================
# 5. PERFORMANCE BENCHMARKS
# =============================================================================
log_section "⚡ PERFORMANCE BENCHMARKS"

# Go benchmarks
if command_exists go; then
    log_step "Running Go benchmarks"
    benchmark_output=$(go test -bench=. -benchmem ./... 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Go benchmarks completed"
        echo "$benchmark_output" | grep "Benchmark" | head -5  # Show top 5 benchmarks
    else
        log_warning "Go benchmarks failed or none found"
    fi
else
    log_warning "Go not found, skipping benchmarks"
fi

# =============================================================================
# 6. BUILD VERIFICATION
# =============================================================================
log_section "🔨 BUILD VERIFICATION"

# Go build
if command_exists go; then
    log_step "Building Go gateway"
    if go build ./cmd/gateway >/dev/null 2>&1; then
        log_success "Go gateway builds successfully"
        # Clean up binary
        rm -f gateway
    else
        log_error "Go gateway build failed"
        ((TOTAL_ERRORS++))
    fi
else
    log_warning "Go not found, skipping Go build"
fi

# Python package build
if command_exists pip; then
    log_step "Checking Python package installation"
    if pip install -e packages/bifrost-core --quiet >/dev/null 2>&1; then
        log_success "Python bifrost-core package installs successfully"
    else
        log_error "Python bifrost-core package installation failed"
        ((TOTAL_ERRORS++))
    fi
    
    if pip install -e packages/bifrost --quiet >/dev/null 2>&1; then
        log_success "Python bifrost package installs successfully"
    else
        log_error "Python bifrost package installation failed"
        ((TOTAL_ERRORS++))
    fi
else
    log_warning "pip not found, skipping Python package checks"
fi

# TypeScript build (if VS Code extension exists)
if [ -d "vscode-extension" ] && command_exists npm; then
    log_step "Building TypeScript extension"
    cd vscode-extension
    if npm run compile --silent >/dev/null 2>&1; then
        log_success "TypeScript extension builds successfully"
    else
        log_error "TypeScript extension build failed"
        ((TOTAL_ERRORS++))
    fi
    cd ..
fi

# =============================================================================
# 7. DOCKER BUILD (if Dockerfile exists)
# =============================================================================
if [ -f "Dockerfile" ] && command_exists docker; then
    log_section "🐳 DOCKER BUILD"
    log_step "Building Docker image"
    if docker build -t bifrost-check . >/dev/null 2>&1; then
        log_success "Docker image builds successfully"
        # Clean up image
        docker rmi bifrost-check >/dev/null 2>&1 || true
    else
        log_error "Docker build failed"
        ((TOTAL_ERRORS++))
    fi
fi

# =============================================================================
# 8. FINAL SUMMARY
# =============================================================================
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

log_section "📊 FINAL SUMMARY"

echo -e "\n${PURPLE}Quality Check Results:${NC}"
echo "======================="

# Test summary
if [[ $TOTAL_TESTS -gt 0 ]]; then
    PASSED_TESTS=$((TOTAL_TESTS - FAILED_TESTS))
    echo -e "Tests:     ${GREEN}$PASSED_TESTS passed${NC}, ${RED}$FAILED_TESTS failed${NC} (Total: $TOTAL_TESTS)"
else
    echo -e "Tests:     ${YELLOW}No tests found${NC}"
fi

# Issue summary
if [[ $TOTAL_ERRORS -eq 0 && $TOTAL_WARNINGS -eq 0 ]]; then
    echo -e "Issues:    ${GREEN}No issues found! 🎉${NC}"
    exit_code=0
elif [[ $TOTAL_ERRORS -eq 0 ]]; then
    echo -e "Issues:    ${YELLOW}$TOTAL_WARNINGS warnings${NC}, ${GREEN}0 errors${NC}"
    exit_code=0
else
    echo -e "Issues:    ${RED}$TOTAL_ERRORS errors${NC}, ${YELLOW}$TOTAL_WARNINGS warnings${NC}"
    exit_code=1
fi

echo -e "Duration:  ${CYAN}${DURATION}s${NC}"

# Recommendations
echo -e "\n${PURPLE}Recommendations:${NC}"
echo "=================="

if [[ $TOTAL_ERRORS -gt 0 ]]; then
    echo -e "${RED}• Fix $TOTAL_ERRORS critical errors before proceeding${NC}"
fi

if [[ $TOTAL_WARNINGS -gt 0 ]]; then
    echo -e "${YELLOW}• Consider addressing $TOTAL_WARNINGS warnings${NC}"
fi

if [[ $FAILED_TESTS -gt 0 ]]; then
    echo -e "${RED}• Fix $FAILED_TESTS failing tests${NC}"
fi

if [[ $TOTAL_ERRORS -eq 0 && $TOTAL_WARNINGS -eq 0 && $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}• Code quality looks excellent! Ready for commit/deploy 🚀${NC}"
fi

# Tool installation recommendations
echo -e "\n${PURPLE}Missing Tools (install for better coverage):${NC}"
! command_exists golangci-lint && echo -e "${YELLOW}• golangci-lint (Go linting): go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest${NC}"
! command_exists ruff && echo -e "${YELLOW}• ruff (Python): pip install ruff${NC}"
! command_exists mypy && echo -e "${YELLOW}• mypy (Python type checking): pip install mypy${NC}"
! command_exists gosec && echo -e "${YELLOW}• gosec (Go security): go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest${NC}"
! command_exists govulncheck && echo -e "${YELLOW}• govulncheck (Go vulnerabilities): go install golang.org/x/vuln/cmd/govulncheck@latest${NC}"
! command_exists bandit && echo -e "${YELLOW}• bandit (Python security): pip install bandit${NC}"
! command_exists markdownlint && echo -e "${YELLOW}• markdownlint (Markdown): npm install -g markdownlint-cli${NC}"
! command_exists pytest && echo -e "${YELLOW}• pytest (Python testing): pip install pytest${NC}"

echo -e "\n${BLUE}Check complete! Exit code: $exit_code${NC}"
exit $exit_code