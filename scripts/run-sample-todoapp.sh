#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}==== Nanostore Sample Todo Application Runner ====${NC}"
echo ""

# Check if we're in the correct directory
if [ ! -f "go.mod" ] || ! grep -q "nanostore" go.mod; then
    echo -e "${RED}Error: This script must be run from the nanostore root directory${NC}"
    exit 1
fi

# Navigate to samples directory
echo -e "${YELLOW}📁 Navigating to samples/todos directory...${NC}"
cd samples/todos

# Clean up any existing test files
echo -e "${YELLOW}🧹 Cleaning up previous test files...${NC}"
rm -f test_todos.json *.json 2>/dev/null || true

# Ensure dependencies are available
echo -e "${YELLOW}📦 Ensuring dependencies are up to date...${NC}"
go mod tidy

# Run the sample application
echo -e "${GREEN}🚀 Running todo validation application...${NC}"
echo ""

go run .

echo ""
echo -e "${GREEN}✅ Sample application completed successfully!${NC}"
echo ""
echo -e "${BLUE}This validates that the JSON store implementation:${NC}"
echo -e "  • Maintains hierarchical ID generation"
echo -e "  • Preserves all document fields (dimension and non-dimension)"
echo -e "  • Supports transparent filtering and ordering"
echo -e "  • Handles status transitions and prefixes correctly"
echo -e "  • Provides type-safe declarative API operations"
echo ""
echo -e "${YELLOW}💡 To run again: ./scripts/run-sample-todoapp.sh${NC}"