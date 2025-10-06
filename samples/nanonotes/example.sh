#!/usr/bin/env bash
set -euo pipefail

# Colors
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

echo -e "${BOLD}Nano-DB Example Scripts${RESET}"
echo
echo "Available examples:"
echo -e "  ${BOLD}./example_1.sh${RESET} - Basic CRUD operations and queries"
echo -e "  ${BOLD}./example_2.sh${RESET} - Bulk operations and advanced queries"
echo
echo -e "${DIM}Run any example script to see nano-db in action with SQL query logging.${RESET}"
echo -e "${DIM}All examples use environment variables for configuration.${RESET}"
echo
echo "To run an example:"
echo "  ./example_1.sh"
echo
echo "Environment variables used:"
echo -e "  ${DIM}NANOSTORE_DB         - Database file path${RESET}"
echo -e "  ${DIM}NANOSTORE_TYPE       - Document type (Note)${RESET}"
echo -e "  ${DIM}NANOSTORE_LOG_QUERIES - Enable query logging (true)${RESET}"
echo