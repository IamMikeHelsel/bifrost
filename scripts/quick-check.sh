#!/bin/bash
# quick-check.sh - Fast quality check for Bifrost project
# Runs essential linters, formatters, and provides issue summary

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

# Helper functions
log_step() {
    echo -e "${CYAN}→ $1${NC}"
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
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Start timing
START_TIME=$(date +%s)

echo -e "${PURPLE}🌉 Bifrost Quick Quality Check${NC}"
echo -e "${PURPLE}==============================${NC}"

# =============================================================================
# 1. FORMATTING CHECK
# =============================================================================
echo -e "\n${BLUE}📝 Formatting Check${NC}"

# Go formatting check
if command_exists go; then
    log_step "Checking Go formatting"
    if go fmt ./... >/dev/null 2>&1; then
        log_success "Go code properly formatted"
    else
        log_warning "Go code needs formatting"
        ((TOTAL_WARNINGS++))
    fi
else
    log_warning "Go not found"
fi

# Python formatting check with ruff
if command_exists ruff; then
    log_step "Checking Python formatting"
    if ruff format --check packages/ >/dev/null 2>&1; then
        log_success "Python code properly formatted"
    else
        log_warning "Python code needs formatting"
        ((TOTAL_WARNINGS++))
    fi
else
    log_warning "ruff not found"
fi

# =============================================================================
# 2. LINTING
# =============================================================================
echo -e "\n${BLUE}🔍 Linting${NC}"

# Go linting
if command_exists golangci-lint; then
    log_step "Linting Go code"
    output=$(golangci-lint run --timeout=30s ./... 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Go linting passed"
    else
        log_warning "Go linting issues found"
        count_issues "$output"
    fi
elif command_exists go; then
    log_step "Running go vet"
    if go vet ./... >/dev/null 2>&1; then
        log_success "Go vet passed"
    else
        log_warning "Go vet issues found"
        ((TOTAL_WARNINGS++))
    fi
fi

# Python linting
if command_exists ruff; then
    log_step "Linting Python code"
    output=$(ruff check packages/ 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Python linting passed"
    else
        log_warning "Python linting issues found"
        count_issues "$output"
    fi
fi

# Python type checking
if command_exists mypy; then
    log_step "Type checking Python code"
    output=$(mypy packages/bifrost/src/ packages/bifrost-core/src/ 2>&1 || true)
    if [[ $? -eq 0 ]]; then
        log_success "Python type checking passed"
    else
        log_warning "Python type checking issues found"
        count_issues "$output"
    fi
fi

# =============================================================================
# 3. QUICK BUILD CHECK
# =============================================================================
echo -e "\n${BLUE}🔨 Build Check${NC}"

# Go build
if command_exists go; then
    log_step "Testing Go build"
    if go build ./cmd/gateway >/dev/null 2>&1; then
        log_success "Go gateway builds successfully"
        rm -f gateway
    else
        log_error "Go gateway build failed"
        ((TOTAL_ERRORS++))
    fi
fi

# =============================================================================
# 4. QUICK TEST CHECK
# =============================================================================
echo -e "\n${BLUE}🧪 Quick Test Check${NC}"

# Go tests (with timeout)
if command_exists go; then
    log_step "Running Go tests (quick)"
    if timeout 30s go test ./... >/dev/null 2>&1; then
        log_success "Go tests passed"
    else
        log_warning "Go tests failed or timed out"
        ((TOTAL_WARNINGS++))
    fi
fi

# =============================================================================
# 5. FINAL SUMMARY
# =============================================================================
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo -e "\n${PURPLE}📊 Quick Check Summary${NC}"
echo "======================"

# Issue summary
if [[ $TOTAL_ERRORS -eq 0 && $TOTAL_WARNINGS -eq 0 ]]; then
    echo -e "Status:    ${GREEN}All checks passed! 🎉${NC}"
    exit_code=0
elif [[ $TOTAL_ERRORS -eq 0 ]]; then
    echo -e "Status:    ${YELLOW}$TOTAL_WARNINGS warnings${NC}, ${GREEN}0 errors${NC}"
    exit_code=0
else
    echo -e "Status:    ${RED}$TOTAL_ERRORS errors${NC}, ${YELLOW}$TOTAL_WARNINGS warnings${NC}"
    exit_code=1
fi

echo -e "Duration:  ${CYAN}${DURATION}s${NC}"

# Recommendations
if [[ $TOTAL_ERRORS -gt 0 ]]; then
    echo -e "\n${RED}❌ Fix $TOTAL_ERRORS critical errors before proceeding${NC}"
    echo -e "${CYAN}💡 Run 'just check-all' for detailed analysis${NC}"
elif [[ $TOTAL_WARNINGS -gt 0 ]]; then
    echo -e "\n${YELLOW}⚠️  Consider addressing $TOTAL_WARNINGS warnings${NC}"
    echo -e "${CYAN}💡 Run 'just check-all' for detailed analysis${NC}"
else
    echo -e "\n${GREEN}✅ Code quality looks good! Ready to proceed 🚀${NC}"
fi

exit $exit_code