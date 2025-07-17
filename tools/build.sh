#!/bin/bash
# Build all tools in the tools directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Building Xatu development tools..."

# Build ethstats-client with tools tag
echo "→ Building ethstats-client..."
cd "$PROJECT_ROOT"
go build -tags tools -o bin/ethstats-client ./tools/ethstats-client

echo "✓ Tools built successfully in bin/"
echo ""
echo "Available tools:"
echo "  - bin/ethstats-client"