#!/usr/bin/env bash
set -euo pipefail

# Source the base setup
source "$(dirname "$0")/example_base.sh"

echo "=== Example 2: Bulk Operations and Advanced Queries ==="
echo

# Create multiple notes for bulk operations
echo "Setting up test data..."
echo

# Create 10 work notes
for i in {1..10}; do
    run_query "Creating work note #$i" \
        $NANO_DB create \
        "Work Task $i" \
        --body="Task description for item $i" \
        --category=work \
        --tags="task,bulk-test,priority-$((i % 3 + 1))"
done

# Create 5 personal notes
for i in {1..5}; do
    run_query "Creating personal note #$i" \
        $NANO_DB create \
        "Personal Item $i" \
        --body="Personal task $i" \
        --category=personal \
        --tags="home,bulk-test"
done

# Create 5 reference notes
for i in {1..5}; do
    run_query "Creating reference note #$i" \
        $NANO_DB create \
        "Reference Doc $i" \
        --body="Reference material $i" \
        --category=reference \
        --tags="docs,bulk-test"
done

# Show initial count
run_query "Count all notes" \
    $NANO_DB stats

# Bulk queries with complex filters
run_query "Find all bulk-test notes" \
    $NANO_DB list \
    --tags__contains=bulk-test

run_query "Find high priority work tasks" \
    $NANO_DB list \
    --category=work --tags__contains=priority-3

run_query "Find notes with 'Task' in title" \
    $NANO_DB list \
    --title__contains=Task

# Bulk updates using queries
echo "=== Bulk Update Operations ==="
echo

# Update all personal notes to add an 'updated' tag
for i in {1..5}; do
    id=$((10 + i))  # Personal notes start after work notes
    run_query "Update personal note $id" \
        $NANO_DB update $id \
        --tags="home,bulk-test,updated"
done

# Show updated personal notes
run_query "Show updated personal notes" \
    $NANO_DB list \
    --category=personal --tags__contains=updated

# Complex multi-condition queries
echo "=== Complex Multi-Condition Queries ==="
echo

run_query "Find work OR reference notes with bulk-test tag" \
    $NANO_DB list \
    --category=work --or --category=reference --tags__contains=bulk-test

run_query "Find notes starting with 'Work' or 'Personal'" \
    $NANO_DB list \
    --title__startswith=Work --or --title__startswith=Personal

# Bulk delete operations
echo "=== Bulk Delete Operations ==="
echo

# Delete all reference notes
for i in {16..20}; do
    run_query "Delete reference note $i" \
        $NANO_DB delete $i
done

run_query "Verify reference notes deleted" \
    $NANO_DB list \
    --category=reference

# Final statistics
run_query "Final statistics" \
    $NANO_DB stats

# Show remaining data grouped by category
echo "=== Remaining Data by Category ==="
echo

for category in work personal idea reference; do
    echo -e "${DIM}Category: $category${RESET}"
    count=$($NANO_DB list --category=$category 2>/dev/null | grep -c "SimpleID" || echo "0")
    echo -e "${DIM}Count: $count${RESET}"
    echo
done

# Show final database contents
show_db_contents