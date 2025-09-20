#!/bin/bash

# Standard comment for tests that should follow model
STANDARD_COMMENT="// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)
"

# Comment for tests that are exceptions (internal package tests)
INTERNAL_COMMENT="// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This is an internal package test that needs access to unexported types.
// It cannot use the standard fixture approach but should still follow other best practices where possible.
"

# Comment for validation tests
VALIDATION_COMMENT="// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This test validates error conditions and configuration failures.
// It creates fresh stores to test specific validation scenarios.
"

# Comment for utility tests
UTILITY_COMMENT="// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This test validates utility functions and type conversions.
// It doesn't require store operations or fixture data.
"

# Function to add comment to file
add_comment() {
    local file=$1
    local comment=$2
    
    # Check if the comment already exists
    if grep -q "model_test.go" "$file"; then
        echo "Skipping $file - already has model test reference"
        return
    fi
    
    # Get the package line
    package_line=$(grep -n "^package " "$file" | head -1 | cut -d: -f1)
    
    # Create temp file with comment after package declaration
    {
        head -n "$package_line" "$file"
        echo ""
        echo "$comment"
        tail -n +"$((package_line + 1))" "$file"
    } > "$file.tmp"
    
    # Replace original file
    mv "$file.tmp" "$file"
    echo "Updated $file"
}

# Process files based on their type

# Standard tests (using fixture)
for file in /Users/adebert/h/nanostore/nanostore/*_migrated_test.go; do
    [ -f "$file" ] && add_comment "$file" "$STANDARD_COMMENT"
done

# API tests
for file in /Users/adebert/h/nanostore/nanostore/api/*_test.go; do
    [ -f "$file" ] && add_comment "$file" "$STANDARD_COMMENT"
done

# Internal package tests
add_comment "/Users/adebert/h/nanostore/nanostore/command_preprocessor_test.go" "$INTERNAL_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/command_preprocessor_complex_test.go" "$INTERNAL_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/lock_manager_test.go" "$INTERNAL_COMMENT"

# Validation tests
add_comment "/Users/adebert/h/nanostore/nanostore/complex_type_validation_test.go" "$VALIDATION_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/config_test.go" "$VALIDATION_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/config_validation_robustness_test.go" "$VALIDATION_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/pointer_validation_test.go" "$VALIDATION_COMMENT"

# Utility/helper tests
add_comment "/Users/adebert/h/nanostore/nanostore/dimension_test.go" "$UTILITY_COMMENT"

# Regular tests that haven't been migrated yet
add_comment "/Users/adebert/h/nanostore/nanostore/non_dimension_fields_test.go" "$STANDARD_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/non_dimension_filtering_test.go" "$STANDARD_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/transparent_verification_test.go" "$STANDARD_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/typed_store_smart_id_test.go" "$STANDARD_COMMENT"

# ID package tests
add_comment "/Users/adebert/h/nanostore/nanostore/ids/id_generator_test.go" "$STANDARD_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/ids/id_transform_test.go" "$STANDARD_COMMENT"

# Store package tests
add_comment "/Users/adebert/h/nanostore/nanostore/stores/file_lock_test.go" "$INTERNAL_COMMENT"
add_comment "/Users/adebert/h/nanostore/nanostore/stores/persistence_test.go" "$STANDARD_COMMENT"

echo "Done adding test comments"