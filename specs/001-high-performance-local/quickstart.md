# Quickstart Guide: High Performance Local CLI Vectorized Codebase Search

## Overview
This guide will help you get started with the high-performance local CLI vectorized codebase search tool. This tool allows you to quickly search through your local codebase using vectorized search for more relevant results.

## Installation
1. Navigate to your project directory where you want to use the search tool
2. Build the CLI tool:
   ```bash
   go build -o code-search ./src/cli/main.go
   ```

## Basic Usage
1. Initialize the search index in your codebase:
   ```bash
   ./code-search index
   ```
   This will scan your current directory and subdirectories, creating a vectorized index of your code.

2. Search for code patterns:
   ```bash
   ./code-search search "authentication function"
   ```
   This will search your indexed codebase for code related to authentication functions.

## Advanced Usage
1. Search with specific file filtering:
   ```bash
   ./code-search search --file-pattern="*.go" "database query"
   ```

2. Limit the number of results:
   ```bash
   ./code-search search --max-results=5 "error handling"
   ```

3. Search with context included:
   ```bash
   ./code-search search --with-context "user validation"
   ```

## Example Workflow
1. Navigate to your codebase root:
   ```bash
   cd /path/to/your/project
   ```

2. Index your codebase:
   ```bash
   ./code-search index
   ```
   Expected output: "Indexing complete. Indexed X files in Y seconds."

3. Search for specific functionality:
   ```bash
   ./code-search search "calculate tax"
   ```
   Expected output:
   ```
   Found 3 results:
   
   1. ./src/calculator/tax.go:24-28
      func calculateTax(amount float64) float64 {
          return amount * 0.08
      }
   
   2. ./src/models/invoice.go:45-52
      // CalculateTax computes the tax for this invoice
      func (i *Invoice) CalculateTax() {
          i.Tax = i.Subtotal * 0.08
      }
   
   3. ./src/handlers/billing.go:67-72
      // Process tax calculation for order
      tax := calculateTax(order.Total)
   ```

## Verification Steps
To verify the tool is working correctly:

1. **Index Creation**: After running `code-search index`, confirm that an index file is created in your project directory.

2. **Search Results**: Perform a search for known code patterns in your codebase and verify that relevant results are returned.

3. **Performance**: Verify that search results are returned within the performance requirements (under 2 seconds for typical searches).

4. **Context Accuracy**: Ensure the file paths, line numbers, and code context in search results accurately reflect the actual code.

## Troubleshooting
- If the index command fails, ensure you have read permissions for all files in the directory.
- If search returns no results, make sure you ran the index command first.
- For large codebases, indexing may take several seconds to complete.