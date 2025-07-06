# Automatic Markdown Formatting Setup

This project has multiple ways to automatically format markdown files. Choose the approach that works best for your workflow.

## 🪝 Option 1: Pre-commit Hooks (Recommended)

**Automatically formats files on every commit**

### Setup

```bash
# One-time setup
just dev-setup  # This installs pre-commit hooks automatically

# Or manually
just install-hooks
```

### How it works

- Every time you `git commit`, pre-commit automatically:
  - Formats Python code with `ruff`
  - Formats Rust code with `rustfmt`
  - Formats markdown with `mdformat`
  - Fixes trailing whitespace and line endings
  - Validates YAML/TOML files

### Manual formatting

```bash
# Format all files now
just fmt-all

# Or run specific formatters
just fmt          # Format code only
mdformat .        # Format markdown only
```

## 💻 Option 2: VS Code Auto-formatting

**Formats files automatically when you save**

### Setup

1. Install recommended extensions (VS Code will prompt you)
1. Settings are already configured in `.vscode/settings.json`

### How it works

- **On Save**: Automatically formats Python, Markdown, TOML, JSON, YAML
- **On Paste**: Formats pasted content
- **Code Actions**: Organizes imports and fixes issues

### Key extensions

- `charliermarsh.ruff` - Python formatting/linting
- `DavidAnson.vscode-markdownlint` - Markdown formatting
- `tamasfe.even-better-toml` - TOML formatting

## 👀 Option 3: Watch Mode

**Continuously watches and formats markdown files**

### Usage

```bash
# Start watching (runs in foreground)
just watch-md

# Files are automatically formatted when changed
# Press Ctrl+C to stop
```

Perfect for when you're writing documentation and want instant formatting.

## 🤖 Option 4: GitHub Actions

**Automatically formats code in pull requests**

### How it works

- Runs on every pull request
- If formatting is needed, automatically commits the changes
- Configured in `.github/workflows/format.yml`

### Benefits

- Ensures all code in the repo stays formatted
- No manual intervention needed
- Works for external contributors

## 🛠 Custom Configuration

### Markdown formatting options

Edit `.pre-commit-config.yaml` to customize mdformat:

```yaml
- repo: https://github.com/executablebooks/mdformat
  rev: 0.7.17
  hooks:
    - id: mdformat
      additional_dependencies:
        - mdformat-gfm        # GitHub Flavored Markdown
        - mdformat-tables     # Table formatting
        - mdformat-footnote   # Footnote support
```

### VS Code settings

Edit `.vscode/settings.json` to customize editor behavior:

```json
{
  "[markdown]": {
    "editor.wordWrap": "bounded",
    "editor.wordWrapColumn": 80,
    "editor.formatOnSave": true
  }
}
```

## 🚀 Quick Start

```bash
# Set up everything
just dev-setup

# Make some changes to markdown files
echo "# Test" >> test.md

# Commit (formatting happens automatically)
git add test.md
git commit -m "Add test file"

# Or format manually
just fmt
```

## 📋 Commands Summary

| Command | Description |
|---------|-------------|
| `just dev-setup` | One-time setup (includes hooks) |
| `just fmt` | Format all code |
| `just fmt-all` | Run pre-commit on all files |
| `just watch-md` | Watch and auto-format markdown |
| `mdformat .` | Format markdown only |
| `mdformat --check .` | Check if markdown needs formatting |

## 🔧 Troubleshooting

### Pre-commit not working?

```bash
# Reinstall hooks
just install-hooks

# Check hook status
uv run pre-commit --version
```

### VS Code not formatting?

1. Check if extensions are installed
1. Restart VS Code
1. Check Output panel for errors

### Watch mode not working?

```bash
# Make sure mdformat is installed
uv run mdformat --version

# Check file permissions
ls -la .last-format
```

Now your markdown files will stay consistently formatted! 🎉
