#!/bin/bash

# Better script to fix performance test files

find tests/integration/resources/TestIndexingPerformance -name "*.go" | while read file; do
    echo "Processing $file"

    # Fix malformed import statements
    sed -i '' 's/"import"/import (/g' "$file"
    sed -i '' 's/" "/)/g' "$file"

    # Fix empty import blocks
    sed -i '' '/import ()/,/)/ {
        /import ()/d
        /^)/d
    }' "$file"

    # Remove any remaining empty lines at the top after package declaration
    sed -i '' '/^package /n; /^$/d' "$file"

    echo "Fixed import syntax in $file"
done

echo "Fixed all performance test files"