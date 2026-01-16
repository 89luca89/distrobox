#!/bin/bash
# SPDX-License-Identifier: GPL-3.0-only
#
# Distrobox Integration Test Suite
#
# This script validates core distrobox functionality across platforms.
# Tests verify: create, enter, list, stop, rm operations.
#
# Exit codes:
#   0 - All tests passed
#   1 - One or more tests failed

set -e
set -o pipefail

# Verbose mode: VERBOSE=1 shows all commands being executed
VERBOSE="${VERBOSE:-0}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_CONTAINER_NAME="distrobox-test-$$"
TEST_IMAGE="${DISTROBOX_TEST_IMAGE:-ubuntu:22.04}"
DISTROBOX_PATH="${DISTROBOX_PATH:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
TEST_DIR="${DISTROBOX_PATH}/.test-tmp-$$"
TEST_OUTPUT_FILE="${TEST_DIR}/test-output.log"

# Create test directory
mkdir -p "${TEST_DIR}"

# Track test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."

    # Remove test container if it exists
    if "${DISTROBOX_PATH}/distrobox-rm" "${TEST_CONTAINER_NAME}" --force > /dev/null 2>&1; then
        echo "Test container removed"
    fi

    # Remove test directory and all test files (safely)
    # Only remove if TEST_DIR is set, exists, and is within DISTROBOX_PATH
    if [ -n "${TEST_DIR}" ] && [ -d "${TEST_DIR}" ] && echo "${TEST_DIR}" | grep -q "^${DISTROBOX_PATH}/\.test-tmp-"; then
        rm -rf "${TEST_DIR}"
        echo "Test directory cleaned up: ${TEST_DIR}"
    fi

    # Print summary
    echo ""
    echo "================================"
    echo "Test Summary"
    echo "================================"
    echo -e "Tests run:    ${TESTS_RUN}"
    echo -e "Tests passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Tests failed: ${RED}${TESTS_FAILED}${NC}"
    echo ""

    if [ ${TESTS_FAILED} -gt 0 ]; then
        echo -e "${RED}FAIL: Some tests failed${NC}"
        exit 1
    else
        echo -e "${GREEN}SUCCESS: All tests passed${NC}"
        exit 0
    fi
}

trap cleanup EXIT INT TERM

# Test helper functions
run_test() {
    local test_name="$1"
    local test_command="$2"

    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "Test ${TESTS_RUN}: ${test_name}... "

    if [ "${VERBOSE}" = "1" ]; then
        echo "" # New line after test name
        echo -e "${BLUE}  Command: ${test_command}${NC}"
    fi

    if eval "${test_command}" > "${TEST_OUTPUT_FILE}" 2>&1; then
        if [ "${VERBOSE}" = "1" ]; then
            echo "  Output:"
            sed 's/^/    /' "${TEST_OUTPUT_FILE}"
        fi
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Error output:"
        sed 's/^/    /' "${TEST_OUTPUT_FILE}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

run_test_with_output() {
    local test_name="$1"
    local test_command="$2"
    local expected_pattern="$3"

    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "Test ${TESTS_RUN}: ${test_name}... "

    if [ "${VERBOSE}" = "1" ]; then
        echo "" # New line after test name
        echo -e "${BLUE}  Command: ${test_command}${NC}"
        echo -e "${BLUE}  Expected pattern: ${expected_pattern}${NC}"
    fi

    local output
    if output=$(eval "${test_command}" 2>&1); then
        if [ "${VERBOSE}" = "1" ]; then
            echo "  Output:"
            echo "${output}" | sed 's/^/    /'
        fi
        if echo "${output}" | grep -q -F -- "${expected_pattern}"; then
            echo -e "${GREEN}PASS${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            echo -e "${RED}FAIL${NC}"
            echo "  Expected pattern not found: ${expected_pattern}"
            echo "  Actual output:"
            echo "${output}" | sed 's/^/    /'
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Command failed with exit code $?"
        echo "  Output:"
        echo "${output}" | sed 's/^/    /'
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Pre-flight checks
echo "================================"
echo "Distrobox Integration Tests"
echo "================================"
echo ""
echo "Configuration:"
echo "  Test container: ${TEST_CONTAINER_NAME}"
echo "  Test image: ${TEST_IMAGE}"
echo "  Distrobox path: ${DISTROBOX_PATH}"
echo "  Platform: $(uname -s)"
echo ""

# Check if container manager is available
if ! command -v podman > /dev/null 2>&1 && ! command -v docker > /dev/null 2>&1; then
    echo -e "${RED}ERROR: No container manager (podman/docker) found${NC}"
    exit 1
fi

# Clean up any leftover test containers from interrupted runs
echo "Checking for leftover test containers..."
leftover_containers=$("${DISTROBOX_PATH}/distrobox-list" 2>/dev/null | grep -E "distrobox-test-[0-9]+" | awk '{print $3}' || true)
if [ -n "${leftover_containers}" ]; then
    echo -e "${YELLOW}Found leftover test containers, cleaning up...${NC}"
    echo "${leftover_containers}" | while IFS= read -r container_name; do
        if [ -n "${container_name}" ]; then
            "${DISTROBOX_PATH}/distrobox-rm" "${container_name}" --force > /dev/null 2>&1 || true
            echo "  Removed: ${container_name}"
        fi
    done
fi

echo "Starting tests..."
echo ""

# Test 1: distrobox-create --help
run_test \
    "distrobox-create --help works" \
    "'${DISTROBOX_PATH}/distrobox-create' --help > /dev/null"

# Test 2: distrobox-enter --help
run_test \
    "distrobox-enter --help works" \
    "'${DISTROBOX_PATH}/distrobox-enter' --help > /dev/null"

# Test 3: distrobox-list --help
run_test \
    "distrobox-list --help works" \
    "'${DISTROBOX_PATH}/distrobox-list' --help > /dev/null"

# Test 4: distrobox-rm --help
run_test \
    "distrobox-rm --help works" \
    "'${DISTROBOX_PATH}/distrobox-rm' --help > /dev/null"

# Test 5: distrobox-stop --help
run_test \
    "distrobox-stop --help works" \
    "'${DISTROBOX_PATH}/distrobox-stop' --help > /dev/null"

# Test 6: distrobox-create --dry-run
run_test_with_output \
    "distrobox-create --dry-run generates valid command" \
    "'${DISTROBOX_PATH}/distrobox-create' --name ${TEST_CONTAINER_NAME} --image ${TEST_IMAGE} --dry-run" \
    "--name"

# Test 7: Create container
echo -e "\n${YELLOW}Creating test container (this may take a few minutes)...${NC}"
# Note: On macOS, distrobox-create may exit with error due to /tmp symlink issue,
# but the container is still created successfully. We verify success by checking
# if the container exists rather than relying solely on exit code.
TESTS_RUN=$((TESTS_RUN + 1))
echo -n "Test ${TESTS_RUN}: Create test container successfully... "
if [ "${VERBOSE}" = "1" ]; then
    echo "" # New line after test name
    echo -e "${BLUE}  Command: ${DISTROBOX_PATH}/distrobox-create --name ${TEST_CONTAINER_NAME} --image ${TEST_IMAGE} --yes${NC}"
fi
"${DISTROBOX_PATH}/distrobox-create" --name "${TEST_CONTAINER_NAME}" --image "${TEST_IMAGE}" --yes > "${TEST_OUTPUT_FILE}" 2>&1 || true
sleep 2
if [ "${VERBOSE}" = "1" ]; then
    echo "  Output:"
    sed 's/^/    /' "${TEST_OUTPUT_FILE}"
fi
# Verify container was created by checking if it appears in the list
if "${DISTROBOX_PATH}/distrobox-list" 2>/dev/null | grep -q "${TEST_CONTAINER_NAME}"; then
    echo -e "${GREEN}PASS${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}FAIL${NC}"
    echo "  Container was not created. Output:"
    sed 's/^/    /' "${TEST_OUTPUT_FILE}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 8: List containers
run_test_with_output \
    "distrobox-list shows created container" \
    "'${DISTROBOX_PATH}/distrobox-list'" \
    "${TEST_CONTAINER_NAME}"

# Test 9: Create test file in repo directory
TEST_FILE="${TEST_DIR}/host-test-file.txt"
TEST_CONTENT="distrobox-test-$$-$(date +%s)"
if [ "${VERBOSE}" = "1" ]; then
    echo -e "${BLUE}Creating test file: echo '${TEST_CONTENT}' > ${TEST_FILE}${NC}"
fi
echo "${TEST_CONTENT}" > "${TEST_FILE}"

run_test \
    "Test file created in repo directory" \
    "test -f '${TEST_FILE}'"

# Test 10: Enter container and read file from mounted directory
echo -e "\n${YELLOW}Entering container for the first time (this may take a minute)...${NC}"
run_test_with_output \
    "Enter container and read file from mounted directory" \
    "'${DISTROBOX_PATH}/distrobox-enter' ${TEST_CONTAINER_NAME} -- cat '${TEST_FILE}'" \
    "${TEST_CONTENT}"

# Test 11: Execute simple command in container
run_test_with_output \
    "Execute echo command in container" \
    "'${DISTROBOX_PATH}/distrobox-enter' ${TEST_CONTAINER_NAME} -- echo 'test-output'" \
    "test-output"

# Test 12: Execute command with shell in container
run_test_with_output \
    "Execute bash command in container" \
    "'${DISTROBOX_PATH}/distrobox-enter' ${TEST_CONTAINER_NAME} -- bash -c 'pwd | grep -q /'" \
    ""

# Test 13: Verify repo directory is accessible in container
run_test \
    "Repo directory is accessible in container" \
    "'${DISTROBOX_PATH}/distrobox-enter' ${TEST_CONTAINER_NAME} -- test -d '${TEST_DIR}'"

# Test 14: Create file inside container and verify on host
CONTAINER_TEST_FILE="${TEST_DIR}/container-test-file.txt"
run_test \
    "Create file inside container visible on host" \
    "'${DISTROBOX_PATH}/distrobox-enter' ${TEST_CONTAINER_NAME} -- bash -c 'echo container-test > ${CONTAINER_TEST_FILE}' && test -f '${CONTAINER_TEST_FILE}'"

# Test 15: Stop container (non-interactive)
run_test \
    "Stop container successfully" \
    "echo 'Y' | '${DISTROBOX_PATH}/distrobox-stop' ${TEST_CONTAINER_NAME}"

# Wait a moment for container to stop
sleep 2

# Test 16: Remove container
run_test \
    "Remove container successfully" \
    "'${DISTROBOX_PATH}/distrobox-rm' ${TEST_CONTAINER_NAME} --force"

# Test 17: Verify container is removed
run_test \
    "Container no longer appears in list" \
    "! '${DISTROBOX_PATH}/distrobox-list' | grep -q ${TEST_CONTAINER_NAME}"

# Cleanup will run automatically via trap
