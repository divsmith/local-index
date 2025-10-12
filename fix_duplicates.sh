#!/bin/bash

# Fix duplicate function declarations

echo "Fixing duplicate function declarations..."

# Remove duplicate contains functions and replace with direct strings.Contains calls
for file in tests/integration/test_project_workflow.go tests/integration/test_multi_project.go; do
    echo "Processing $file"

    # Remove the contains function definition
    sed -i '' '/^\/\/ Helper function to check if a string contains a substring$/,/^func contains(s, substr string) bool {$/,/^}$/d' "$file"

    # Replace all calls to contains() with strings.Contains()
    sed -i '' 's/contains(\([^,]*\), \([^)]*\))/strings.Contains(\1, \2)/g' "$file"

    echo "Fixed contains function in $file"
done

# Rename createProjectStructure in test_multi_project.go to createMultiProjectStructure
echo "Renaming createProjectStructure in test_multi_project.go..."
sed -i '' 's/createProjectStructure(/createMultiProjectStructure(/g' tests/integration/test_multi_project.go

# Rename the function definition
sed -i '' 's/^func createProjectStructure(t \*testing.T, projectDir string) {$/func createMultiProjectStructure(t **testing.T, projectDir string) {/' tests/integration/test_multi_project.go

echo "Fixed all duplicate function declarations"