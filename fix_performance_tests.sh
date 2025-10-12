#!/bin/bash

# Fix all performance test files by removing unused time import and fixing function return mismatches

find tests/integration/resources/TestIndexingPerformance -name "*.go" | while read file; do
    echo "Processing $file"

    # Check if file imports time but doesn't use it
    if grep -q '"time"' "$file" && ! grep -q "time\." "$file"; then
        # Remove the time import
        sed -i '' '/import (/,/)/ {
            /"time"/d
            /^$/d
        }' "$file"

        # Clean up empty import blocks
        sed -i '' '/import ()/d' "$file"

        # Clean up import blocks that now have only one import
        sed -i '' '/import (/,/)/ {
            s/import (/"import"/
            s/)/" "/
        }' "$file"

        echo "Removed unused time import from $file"
    fi

    # Fix the function return mismatch issue
    # Look for lines that assign function return to two variables when function only returns one
    sed -i '' 's/\([a-zA-Z_][a-zA-Z0-9_]*\), err := \([A-Z][a-zA-Z0-9]*\)(/err := \2(/g' "$file"

    # Then fix the usage of the first variable in the next line
    sed -i '' 's/result = append(result, fmt.Sprintf("%d: %s", i, \([a-zA-Z_][a-zA-Z0-9_]*\)))/result = append(result, fmt.Sprintf("%d: %s", i, item))/g' "$file"

done

echo "Fixed all performance test files"