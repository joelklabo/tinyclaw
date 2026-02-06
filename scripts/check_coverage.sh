#!/usr/bin/env bash
set -euo pipefail

# Coverage gate: fails unless total coverage = 100.0% (excluding cmd/)
# Usage: ./scripts/check_coverage.sh

COVER_PROFILE=$(mktemp)
trap 'rm -f "$COVER_PROFILE"' EXIT

# Collect all packages excluding cmd/
PKGS=$(go list ./... | grep -v '/cmd/')

if [ -z "$PKGS" ]; then
  echo "FAIL: no packages found to test"
  exit 1
fi

# Run tests with coverage
go test -race -count=1 -coverprofile="$COVER_PROFILE" $PKGS > /dev/null 2>&1

# Extract total coverage
TOTAL=$(go tool cover -func="$COVER_PROFILE" | grep '^total:' | awk '{print $NF}' | tr -d '%')

if [ -z "$TOTAL" ]; then
  echo "FAIL: could not determine coverage"
  exit 1
fi

echo "Coverage: ${TOTAL}%"

if [ "$TOTAL" != "100.0" ]; then
  echo "FAIL: coverage is ${TOTAL}%, required 100.0%"
  echo ""
  echo "Uncovered lines:"
  go tool cover -func="$COVER_PROFILE" | grep -v '100.0%' | grep -v '^total:'
  exit 1
fi

echo "PASS: 100.0% coverage"
