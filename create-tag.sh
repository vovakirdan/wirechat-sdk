#!/bin/bash
set -e

echo "=== Creating Git Tag for WireChat Go SDK ==="
echo

cd "$(dirname "$0")"

echo "Current directory: $(pwd)"
echo

echo "Step 1: Checking git status..."
git status
echo

echo "Step 2: Staging SDK changes..."
git add wirechat-sdk-go/
echo "✓ Changes staged"
echo

echo "Step 3: Creating commit..."
git commit -m "fix: update Go SDK module path for submodule structure

- Change module path from github.com/vovakirdan/wirechat-sdk-go
  to github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go
- Update all internal imports
- Update examples to use correct import path"
echo "✓ Commit created"
echo

echo "Step 4: Creating tag wirechat-sdk-go/v0.1.0..."
git tag -a wirechat-sdk-go/v0.1.0 -m "Go SDK v0.1.0

Initial release of WireChat Go SDK with:
- WebSocket client for WireChat protocol v1
- Event-driven API (OnMessage, OnUserJoined, OnUserLeft, OnError)
- Room management (Join, Leave)
- Message sending
- JWT authentication support
- Configurable timeouts
"
echo "✓ Tag created"
echo

echo "Step 5: Showing tag info..."
git show wirechat-sdk-go/v0.1.0 --no-patch
echo

echo "Next steps:"
echo "1. Push commit: git push origin main"
echo "2. Push tag:    git push origin wirechat-sdk-go/v0.1.0"
echo
echo "After pushing, users can install with:"
echo "  go get github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go@v0.1.0"
