# gtt

> A modern Git + GitHub CLI that makes branch management a breeze

`gtt` is a command-line tool that enhances your Git workflow with interactive branch visualization, stacked PR support, and seamless GitHub integration.

## Features

- **Interactive Branch Tree** - Visualize and navigate your branches in a beautiful tree structure
- **Stack-aware** - Understands branch relationships, even after merges
- **GitHub Integration** - Create and manage PRs without leaving the terminal
- **Configurable Trunk** - Works with `main`, `develop`, or any branch as your trunk

## Installation

### Using Go

```bash
go install github.com/lehi10/Gitty@latest
```

### From Source

```bash
git clone https://github.com/lehi10/Gitty.git
cd Gitty
go build -o gtt .
sudo mv gtt /usr/local/bin/
```

### Homebrew (coming soon)

```bash
brew install lehi10/tap/gtt
```

## Quick Start

```bash
# Initialize gtt in your repo (select your trunk branch)
gtt init

# Interactively switch branches with tree view
gtt checkout

# Create a pull request
gtt pr
```

## Commands

### `gtt init`

Initialize gtt in your repository. Interactively select your trunk branch (main, develop, etc.).

```bash
gtt init
```

Creates a `.gttconfig` file with your configuration.

### `gtt checkout [branch]`

Switch branches with an interactive tree view. Shows:
- Branch hierarchy
- Current branch (●)
- Merged branches (✓)

```bash
# Interactive mode
gtt checkout

# Direct checkout
gtt checkout feature/my-branch
```

**Controls:**
- `↑/k` - Move up
- `↓/j` - Move down
- `Enter` - Select branch
- `q/Esc` - Cancel

### `gtt status`

Show the working tree status (alias for `git status`).

```bash
gtt status
```

### `gtt pr`

Create a pull request for the current branch.

```bash
# Interactive PR creation
gtt pr

# List open PRs
gtt pr list

# View current branch's PR in browser
gtt pr view
```

**Features:**
- Auto-detects base branch from config
- Auto-pushes if branch isn't on remote
- Interactive title and description input

**Requires:** [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated.

## Configuration

gtt stores configuration in `.gttconfig` at the repository root:

```json
{
  "trunk": "develop"
}
```

## Branch Tree Example

```
○ main
└── ○ develop
    ├── ✓ feature/auth (merged)
    │   ├── ✓ feature/auth-login (merged)
    │   └── ✓ feature/auth-logout (merged)
    ├── ○ feature/dashboard
    │   └── ● feature/dashboard-charts (current)
    └── ✓ feature/api (merged)
```

## Requirements

- Git
- Go 1.21+ (for building from source)
- [GitHub CLI](https://cli.github.com/) (for PR commands)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Inspired by [Graphite](https://graphite.dev/) and the need for better branch visualization in Git workflows.
