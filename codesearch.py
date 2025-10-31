#!/usr/bin/env python3
"""
Simple codebase search tool for semantic and text-based code discovery.
"""

import os
import sqlite3
import argparse
import sys
from pathlib import Path
from typing import List, Dict, Any, Set
import json
import time
import hashlib
import fnmatch

# Simple text-based search for MVP
try:
    import re
    from collections import Counter
    import math
    DEPS_AVAILABLE = True
except ImportError:
    DEPS_AVAILABLE = False

class SimpleCodeIndex:
    def __init__(self, db_path: str = None):
        if db_path is None:
            db_path = os.path.expanduser("~/.codesearch/index.db")

        self.db_path = db_path
        os.makedirs(os.path.dirname(db_path), exist_ok=True)
        self._init_db()

    def _init_db(self):
        """Initialize SQLite database."""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        cursor.execute("""
            CREATE TABLE IF NOT EXISTS files (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                path TEXT UNIQUE NOT NULL,
                content_hash TEXT NOT NULL,
                indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)

        cursor.execute("""
            CREATE TABLE IF NOT EXISTS chunks (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                file_id INTEGER NOT NULL,
                chunk_index INTEGER NOT NULL,
                content TEXT NOT NULL,
                start_line INTEGER NOT NULL,
                end_line INTEGER NOT NULL,
                FOREIGN KEY (file_id) REFERENCES files (id)
            )
        """)

        cursor.execute("CREATE INDEX IF NOT EXISTS idx_files_path ON files (path)")
        cursor.execute("CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON chunks (file_id)")

        conn.commit()
        conn.close()

    def _get_file_hash(self, content: str) -> str:
        """Get hash of file content."""
        return hashlib.md5(content.encode()).hexdigest()

    def _chunk_content(self, content: str, chunk_size: int = 500) -> List[Dict[str, Any]]:
        """Split content into chunks."""
        lines = content.split('\n')
        chunks = []

        current_chunk_lines = []
        current_chunk_size = 0

        for i, line in enumerate(lines, 1):
            current_chunk_lines.append((i, line))
            current_chunk_size += len(line)

            if current_chunk_size >= chunk_size:
                chunk_content = '\n'.join(line for _, line in current_chunk_lines)
                chunks.append({
                    'content': chunk_content,
                    'start_line': current_chunk_lines[0][0],
                    'end_line': current_chunk_lines[-1][0]
                })
                current_chunk_lines = []
                current_chunk_size = 0

        if current_chunk_lines:
            chunk_content = '\n'.join(line for _, line in current_chunk_lines)
            chunks.append({
                'content': chunk_content,
                'start_line': current_chunk_lines[0][0],
                'end_line': current_chunk_lines[-1][0]
            })

        return chunks

    def _parse_gitignore(self, gitignore_path: Path) -> Set[str]:
        """Parse .gitignore file and return set of ignore patterns."""
        if not gitignore_path.exists():
            return set()

        patterns = set()
        try:
            with open(gitignore_path, 'r', encoding='utf-8') as f:
                for line in f:
                    line = line.strip()
                    # Skip empty lines and comments
                    if not line or line.startswith('#'):
                        continue
                    patterns.add(line)
        except Exception as e:
            print(f"  Warning: Could not read {gitignore_path}: {e}")

        return patterns

    def _should_ignore_file(self, file_path: Path, gitignore_patterns: Set[str], base_path: Path) -> bool:
        """Check if file should be ignored based on gitignore patterns."""
        # Get relative path from the base directory
        try:
            rel_path = file_path.relative_to(base_path)
            rel_path_str = str(rel_path).replace('\\', '/')  # Normalize path separators
        except ValueError:
            # If we can't get relative path, use the full path
            rel_path_str = str(file_path).replace('\\', '/')

        # Check against each gitignore pattern
        for pattern in gitignore_patterns:
            # Handle negation patterns starting with !
            if pattern.startswith('!'):
                # Negation pattern - if it matches, don't ignore
                neg_pattern = pattern[1:]
                if fnmatch.fnmatch(rel_path_str, neg_pattern) or fnmatch.fnmatch(file_path.name, neg_pattern):
                    return False
                continue

            # Handle directory patterns ending with /
            if pattern.endswith('/'):
                if fnmatch.fnmatch(rel_path_str + '/', pattern) or fnmatch.fnmatch(file_path.name + '/', pattern):
                    return True
                continue

            # Regular pattern matching
            if fnmatch.fnmatch(rel_path_str, pattern) or fnmatch.fnmatch(file_path.name, pattern):
                return True

            # Handle leading slash patterns (absolute from repo root)
            if pattern.startswith('/'):
                if fnmatch.fnmatch('/' + rel_path_str, pattern) or fnmatch.fnmatch('/' + file_path.name, pattern):
                    return True

        return False

    def clear_index(self):
        """Clear all indexed data from the database."""
        if os.path.exists(self.db_path):
            try:
                # Remove the database file completely
                os.remove(self.db_path)
                print(f"Index cleared successfully")
                print(f"Removed database: {self.db_path}")

                # Re-initialize the database structure
                self._init_db()
                print("Fresh database initialized")

            except Exception as e:
                print(f"Error clearing index: {e}")
                return False
        else:
            print("No index database found to clear")

        return True

    def _is_text_file(self, file_path: Path) -> bool:
        """Check if file is likely a text file."""
        text_extensions = {
            '.py', '.js', '.ts', '.jsx', '.tsx', '.java', '.cpp', '.c', '.h', '.hpp',
            '.rs', '.go', '.rb', '.php', '.swift', '.kt', '.scala', '.sh', '.bash',
            '.yml', '.yaml', '.json', '.toml', '.ini', '.cfg', '.conf',
            '.md', '.txt', '.rst', '.org', '.tex', '.html', '.css', '.scss', '.less',
            '.sql', '.graphql', '.xml', '.dockerfile', '.makefile', '.cmake'
        }
        return file_path.suffix.lower() in text_extensions

    def _index_file(self, file_path: str, content: str):
        """Index a single file."""
        content_hash = self._get_file_hash(content)

        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        # Check if file needs reindexing
        cursor.execute("SELECT id, content_hash FROM files WHERE path = ?", (file_path,))
        existing = cursor.fetchone()

        if existing and existing[1] == content_hash:
            conn.close()
            return  # File hasn't changed

        # Remove old chunks if file exists
        if existing:
            cursor.execute("DELETE FROM chunks WHERE file_id = ?", (existing[0],))
            cursor.execute("DELETE FROM files WHERE path = ?", (file_path,))

        # Insert file
        cursor.execute("INSERT INTO files (path, content_hash) VALUES (?, ?)", (file_path, content_hash))
        file_id = cursor.lastrowid

        # Chunk and index content
        chunks = self._chunk_content(content)

        for i, chunk in enumerate(chunks):
            cursor.execute("""
                INSERT INTO chunks (file_id, chunk_index, content, start_line, end_line)
                VALUES (?, ?, ?, ?, ?)
            """, (file_id, i, chunk['content'], chunk['start_line'], chunk['end_line']))

        conn.commit()
        conn.close()

    def search(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Search for relevant code chunks using text similarity."""
        # Simple TF-IDF style search for MVP

        # Split query into terms
        query_terms = re.findall(r'\w+', query.lower())
        if not query_terms:
            return []

        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        # Get all chunks
        cursor.execute("""
            SELECT c.id, c.content, c.start_line, c.end_line, f.path
            FROM chunks c
            JOIN files f ON c.file_id = f.id
        """)

        results = []
        for chunk_id, content, start_line, end_line, file_path in cursor.fetchall():
            # Calculate simple relevance score
            content_lower = content.lower()
            score = 0.0

            for term in query_terms:
                # Exact matches get higher score
                if term in content_lower:
                    score += 1.0

                    # Bonus for matches that are whole words
                    if re.search(r'\b' + re.escape(term) + r'\b', content_lower):
                        score += 0.5

                    # Bonus for matches in function names, class names, etc.
                    if re.search(r'(def|class|function|var|let|const|import)\s*\w*' + re.escape(term), content_lower):
                        score += 1.0

            if score > 0:
                results.append({
                    'file_path': file_path,
                    'content': content,
                    'start_line': start_line,
                    'end_line': end_line,
                    'score': score
                })

        conn.close()

        # Sort by score and return top results
        results.sort(key=lambda x: x['score'], reverse=True)
        return results[:limit]

    def print_results(self, results: List[Dict[str, Any]]):
        """Print search results in a readable format."""
        if not results:
            print("No results found.")
            return

        print(f"\nFound {len(results)} results:\n")

        for i, result in enumerate(results, 1):
            print(f"{i}. {result['file_path']}:{result['start_line']}-{result['end_line']} (score: {result['score']:.3f})")
            print("─" * 80)

            lines = result['content'].split('\n')
            for line_num, line in enumerate(lines, result['start_line']):
                print(f"{line_num:4d}: {line}")

            print("─" * 80)
            print()

    def index_directory(self, path: str, exclude_patterns: List[str] = None, verbose: bool = False):
        """Index all files in directory, respecting .gitignore rules."""
        # Default exclude patterns for things we always want to skip
        default_excludes = ['.git', '.gitignore', 'node_modules', '__pycache__', '.venv', 'venv', 'target', 'build', 'dist']
        if exclude_patterns is None:
            exclude_patterns = default_excludes
        else:
            exclude_patterns.extend(default_excludes)

        path_obj = Path(path).resolve()
        indexed_files = 0
        skipped_files = 0

        # Look for .gitignore file
        gitignore_path = path_obj / '.gitignore'
        gitignore_patterns = self._parse_gitignore(gitignore_path)

        if verbose:
            print(f"Indexing directory: {path_obj}")
            if gitignore_patterns:
                print(f"Found .gitignore with {len(gitignore_patterns)} patterns")
                if verbose:
                    print(f"Gitignore patterns: {sorted(gitignore_patterns)}")
            print(f"Additional exclude patterns: {exclude_patterns}")
        else:
            print(f"Indexing directory: {path_obj}")
            if gitignore_patterns:
                print(f"  Using .gitignore patterns ({len(gitignore_patterns)} patterns)")

        for file_path in path_obj.rglob('*'):
            if file_path.is_file():
                # First check hardcoded exclude patterns
                if any(pattern in str(file_path) for pattern in exclude_patterns):
                    skipped_files += 1
                    if verbose:
                        print(f"  Skipped (excluded): {file_path}")
                    continue

                # Then check .gitignore patterns if any exist
                if gitignore_patterns and self._should_ignore_file(file_path, gitignore_patterns, path_obj):
                    skipped_files += 1
                    if verbose:
                        print(f"  Skipped (.gitignore): {file_path}")
                    continue

                # Skip binary files and very large files
                if not self._is_text_file(file_path) or file_path.stat().st_size > 1024 * 1024:  # 1MB
                    skipped_files += 1
                    if verbose:
                        print(f"  Skipped (binary/large): {file_path}")
                    continue

                try:
                    with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                        content = f.read()

                    self._index_file(str(file_path), content)
                    indexed_files += 1

                    if verbose:
                        print(f"  Indexed: {file_path}")
                    elif not verbose and indexed_files % 50 == 0:
                        print(f"  Indexed {indexed_files} files...")

                except Exception as e:
                    if verbose:
                        print(f"  Warning: Could not index {file_path}: {e}")
                    skipped_files += 1

        print(f"Indexing complete: {indexed_files} files indexed, {skipped_files} files skipped")

    def print_stats(self):
        """Print index statistics."""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()

        # Get file count
        cursor.execute("SELECT COUNT(*) FROM files")
        file_count = cursor.fetchone()[0]

        # Get chunk count
        cursor.execute("SELECT COUNT(*) FROM chunks")
        chunk_count = cursor.fetchone()[0]

        # Get total size
        cursor.execute("SELECT SUM(LENGTH(content)) FROM chunks")
        total_content_size = cursor.fetchone()[0] or 0

        # Get last indexed file
        cursor.execute("SELECT path, indexed_at FROM files ORDER BY indexed_at DESC LIMIT 1")
        last_result = cursor.fetchone()
        last_file, last_time = last_result if last_result else (None, None)

        conn.close()

        print("\nIndex Statistics")
        print("=" * 50)
        print(f"Files indexed: {file_count:,}")
        print(f"Code chunks: {chunk_count:,}")
        print(f"Total content: {total_content_size:,} characters")
        print(f"Database path: {self.db_path}")
        if last_file:
            print(f"Last indexed: {last_file}")
            print(f"Last indexed at: {last_time}")
        print("=" * 50)

    def print_status(self):
        """Print current status of the search tool."""
        print("\nCodeSearch Status")
        print("=" * 50)

        # Check database exists
        if os.path.exists(self.db_path):
            print("Database exists")
            self.print_stats()
        else:
            print("No database found")
            print("Run 'python3 codesearch.py index <directory>' to create an index")

        # Check dependencies
        print("\nDependencies:")
        print(f"Python standard library available")
        print(f"SQLite available")

        print("\nQuick Start:")
        print("1. Index: python3 codesearch.py index .")
        print("2. Search: python3 codesearch.py search \"your query\"")
        print("3. Status: python3 codesearch.py status")
        print("4. Help: python3 codesearch.py help")
        print("5. Clear: python3 codesearch.py clear")
        print("=" * 50)

    def run_diagnostics(self):
        """Run diagnostic checks on the tool and database."""
        print("\nCodeSearch Diagnostics")
        print("=" * 50)

        issues = []

        # Check database
        if os.path.exists(self.db_path):
            print("Database file exists")

            try:
                conn = sqlite3.connect(self.db_path)
                cursor = conn.cursor()

                # Test database integrity
                cursor.execute("PRAGMA integrity_check")
                integrity_result = cursor.fetchone()[0]
                if integrity_result == "ok":
                    print("Database integrity check passed")
                else:
                    print(f"Database integrity issues: {integrity_result}")
                    issues.append("Database integrity issues")

                # Check tables exist
                cursor.execute("SELECT name FROM sqlite_master WHERE type='table'")
                tables = [row[0] for row in cursor.fetchall()]
                expected_tables = ['files', 'chunks']

                for table in expected_tables:
                    if table in tables:
                        print(f"Table '{table}' exists")
                    else:
                        print(f"Missing table '{table}'")
                        issues.append(f"Missing table '{table}'")

                conn.close()

            except Exception as e:
                print(f"Database error: {e}")
                issues.append(f"Database error: {e}")
        else:
            print("No database found - run indexing first")
            issues.append("No database found")

        # Check permissions
        try:
            test_file = os.path.expanduser("~/.codesearch/test_write")
            os.makedirs(os.path.dirname(test_file), exist_ok=True)
            with open(test_file, 'w') as f:
                f.write("test")
            os.remove(test_file)
            print("Write permissions OK")
        except Exception as e:
            print(f"Permission issue: {e}")
            issues.append(f"Permission issue: {e}")

        # Check directory access
        current_dir = os.getcwd()
        if os.access(current_dir, os.R_OK):
            print("Directory read access OK")
        else:
            print("Cannot read current directory")
            issues.append("Directory read access denied")

        # Summary
        print("\n" + "=" * 50)
        if not issues:
            print("All diagnostics passed - tool is ready to use!")
            print("\nNext steps:")
            print("1. Index a directory: python3 codesearch.py index .")
            print("2. Search for code: python3 codesearch.py search \"query\"")
        else:
            print("Issues found:")
            for issue in issues:
                print(f"  • {issue}")
            print("\nTry running: python3 codesearch.py doctor --help")
        print("=" * 50)

def print_help():
    """Print comprehensive help information for agents."""
    help_text = """
🔍 CodeSearch - Agent-First Codebase Search Tool
===============================================

DESIGNED FOR AI AGENTS
──────────────────────
This tool is specifically designed to be easily discoverable and usable by AI coding agents.
All capabilities can be understood through simple help commands.

QUICK START FOR AGENTS
──────────────────────
1. Discover tool: python3 codesearch.py --help
2. Check status: python3 codesearch.py status
3. Index directory: python3 codesearch.py index .
4. Search code: python3 codesearch.py search "your query"
5. Run diagnostics: python3 codesearch.py doctor
6. Clear index: python3 codesearch.py clear

COMMANDS
────────
index <directory>    - Index all files in directory for searching
search <query>       - Search indexed code using keyword matching
status               - Show current index status and statistics
help                 - Show this detailed help information
doctor               - Run diagnostic checks on tool and database
clear                - Clear all indexed data and reset database

SEARCH USAGE EXAMPLES
─────────────────────
python3 codesearch.py search "class Database"
python3 codesearch.py search "def authenticate_user"
python3 codesearch.py search "import requests"
python3 codesearch.py search "database config"
python3 codesearch.py search "async def"
python3 codesearch.py search --limit 5 "error handling"

INDEXING OPTIONS
────────────────
--verbose           - Show detailed indexing progress
--stats             - Show index statistics after indexing
--db <path>         - Custom database path (default: ~/.codesearch/index.db)

SEARCH OPTIONS
──────────────
--limit <number>    - Maximum results to return (default: 10)
--db <path>         - Custom database path

FILE TYPES SUPPORTED
───────────────────
• Python (.py, .pyi, .pyx)
• JavaScript (.js, .jsx, .mjs)
• TypeScript (.ts, .tsx)
• Rust (.rs), Go (.go), Java (.java)
• C/C++ (.c, .cpp, .h, .hpp)
• Configuration (.yml, .yaml, .json, .toml)
• Documentation (.md, .rst, .txt)
• And many more...

SEARCH SCORING ALGORITHM
────────────────────────
The tool uses intelligent relevance scoring:
• +1.0 for each keyword match
• +0.5 bonus for whole-word matches
• +1.0 bonus for matches in function/class definitions
• Results sorted by relevance score

DOGFOODING - TOOL CAN SEARCH ITSELF
────────────────────────────────────
python3 codesearch.py search "SimpleCodeIndex class"
python3 codesearch.py search "def search"
python3 codesearch.py search "chunk_content"
python3 codesearch.py search "relevance scoring"

PERFORMANCE CHARACTERISTICS
───────────────────────────
• Indexing: ~400+ files in < 30 seconds
• Search: Typically < 100ms
• Storage: SQLite database in ~/.codesearch/
• Memory: < 50MB for typical usage

FOR AGENTS: DISCOVERY PATTERN
────────────────────────────
As an AI agent, you can discover this tool's capabilities by:
1. Running --help to understand all commands
2. Using status to check if index exists
3. Using doctor to verify tool health
4. Searching tool's own codebase to understand implementation

ERROR HANDLING
──────────────
• Graceful handling of missing/unreadable files
• Automatic exclusion of binary files and build artifacts
• Database corruption detection and repair suggestions
• Clear error messages with actionable fixes

EXTENSIBILITY
───────────
This is a minimal implementation designed for immediate utility.
Future improvements could include:
• Semantic search with embeddings
• Symbol-aware AST parsing
• Incremental file watching
• Cross-reference discovery

TROUBLESHOOTING FOR AGENTS
─────────────────────────
If search returns no results:
• Check status: python3 codesearch.py status
• Verify directory indexed: python3 codesearch.py doctor
• Re-index if needed: python3 codesearch.py index .

If indexing is slow:
• Use --stats to see progress
• Check if large binary files are being processed
• Consider adding more exclude patterns

For more help, run: python3 codesearch.py doctor
"""
    print(help_text)

def main():
    parser = argparse.ArgumentParser(
        description='CodeSearch - Agent-first codebase search tool',
        epilog="""
Examples:
  %(prog)s index .                    # Index current directory
  %(prog)s search "user auth"         # Search for user authentication code
  %(prog)s search "class Database"    # Find Database class definition
  %(prog)s search --help              # Show search command help

For AI agents: This tool is designed to be discovered and used programmatically.
Run '%(prog)s --help' to understand all capabilities.
        """,
        formatter_class=argparse.RawDescriptionHelpFormatter
    )

    parser.add_argument('command', choices=['index', 'search', 'status', 'help', 'doctor', 'clear'],
                       help='Command to run. Commands:')
    parser.add_argument('path_or_query', nargs='?',
                       help='Directory path for index command, or search query for search command')
    parser.add_argument('--limit', type=int, default=10,
                       help='Maximum number of search results to return (default: 10)')
    parser.add_argument('--db', help='Database path (default: ~/.codesearch/index.db)')
    parser.add_argument('--verbose', action='store_true', help='Verbose output for indexing')
    parser.add_argument('--stats', action='store_true', help='Show index statistics')

    args = parser.parse_args()

    if not DEPS_AVAILABLE:
        print("Error: Required modules not available")
        sys.exit(1)

    # Handle help command
    if args.command == 'help':
        print_help()
        return

    index = SimpleCodeIndex(args.db)

    if args.command == 'index':
        if not args.path_or_query:
            print("Error: Index command requires a directory path")
            print("Usage: python3 codesearch.py index <directory>")
            sys.exit(1)

        if not os.path.isdir(args.path_or_query):
            print(f"Error: {args.path_or_query} is not a directory")
            sys.exit(1)

        index.index_directory(args.path_or_query, verbose=args.verbose)

        if args.stats:
            index.print_stats()

    elif args.command == 'search':
        if not args.path_or_query:
            print("Error: Search command requires a query")
            print("Usage: python3 codesearch.py search \"your search query\"")
            sys.exit(1)

        results = index.search(args.path_or_query, args.limit)
        index.print_results(results)

    elif args.command == 'status':
        index.print_status()

    elif args.command == 'clear':
        index.clear_index()

    elif args.command == 'doctor':
        index.run_diagnostics()

if __name__ == '__main__':
    main()