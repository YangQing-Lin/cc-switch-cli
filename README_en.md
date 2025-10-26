# cc-switch-cli

A lightweight command-line tool for managing multiple Claude API configurations with quick switching capabilities.

## Features

- 🖥️ **Interactive TUI** - Modern terminal user interface powered by Bubble Tea with keyboard navigation and visual operations
- 🔄 **Quick Switch** - Switch between different Claude API configurations with a single command
- 📝 **Configuration Management** - Add, delete, and view multiple API configurations
- 🔐 **Secure Storage** - API tokens are masked when displayed, configuration files are permission-protected
- 🌍 **Cross-Platform** - Supports Windows, macOS, Linux, and other operating systems
- 💡 **Interactive Input** - Supports both command-line arguments and interactive prompts
- 🎨 **User-Friendly** - Clear list display with intuitive status indicators
- 🎯 **Multi-App Support** - Manage both Claude Code and Codex CLI configurations

## Installation

### Build from Source

Requires Go 1.25.1 or higher:

```bash
git clone https://github.com/YangQing-Lin/cc-switch-cli.git
cd cc-switch-cli

# Windows
go build -o ccs.exe

# macOS / Linux
go build -o ccs
```

### Download Pre-built Binaries

Download the appropriate pre-compiled binary for your operating system from the [Releases](https://github.com/YangQing-Lin/cc-switch-cli/releases) page.

### Configure Environment Variables

To use the `ccs` command from any directory, add it to your system's PATH:

#### Windows

**Method 1: Via PowerShell**
```powershell
# Move binary to user directory
mkdir -Force $env:USERPROFILE\bin
move ccs.exe $env:USERPROFILE\bin\

# Add to PATH (current session)
$env:Path += ";$env:USERPROFILE\bin"

# Permanently add to PATH
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\bin", "User")
```

**Method 2: Via System Settings**
1. Copy `ccs.exe` to a directory, e.g., `C:\Program Files\ccs\`
2. Right-click "This PC" → "Properties" → "Advanced system settings"
3. Click "Environment Variables"
4. In "User variables", find `Path` and click "Edit"
5. Click "New" and add `C:\Program Files\ccs\`
6. Click "OK" to save

#### macOS

```bash
# Move binary to /usr/local/bin
sudo mv ccs /usr/local/bin/

# Or move to user directory
mkdir -p ~/bin
mv ccs ~/bin/

# Add to PATH (if using ~/bin)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc   # zsh
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc  # bash

# Reload configuration
source ~/.zshrc  # or source ~/.bashrc
```

#### Linux

```bash
# Method 1: System-wide installation (requires sudo)
sudo mv ccs /usr/local/bin/
sudo chmod +x /usr/local/bin/ccs

# Method 2: User-level installation
mkdir -p ~/.local/bin
mv ccs ~/.local/bin/
chmod +x ~/.local/bin/ccs

# Add to PATH (if ~/.local/bin is not in PATH)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Verify installation:
```bash
ccs version
```

### Version Updates and Recompiling

When a new version is released, update by following these steps:

```bash
# 1. Navigate to project directory
cd cc-switch-cli

# 2. Pull latest code
git pull origin main

# 3. Recompile
# Windows
go build -o ccs.exe

# macOS / Linux
go build -o ccs

# 4. Replace old version (if environment variable is configured)
# Windows (PowerShell)
move -Force ccs.exe $env:USERPROFILE\bin\ccs.exe

# macOS / Linux
sudo mv ccs /usr/local/bin/ccs  # System-wide
# or
mv ccs ~/.local/bin/ccs  # User-level

# 5. Verify new version
ccs version
```

## Usage

### Interactive TUI Interface (Recommended)

Launch the interactive terminal user interface:

```bash
ccs
# Or explicitly specify
ccs ui
```

**TUI Features:**

- 📋 **Visual List** - Display all configurations clearly at a glance
- ⌨️ **Keyboard Navigation** - Use arrow keys to select configurations
- ✏️ **Instant Editing** - Press `e` to quickly edit a configuration
- ➕ **Quick Add** - Press `a` to add a new configuration
- 🗑️ **Safe Delete** - Press `d` to delete a configuration (with confirmation)
- 🔄 **One-Key Switch** - Press `Enter` to switch to the selected configuration
- 🎨 **Friendly Interface** - Beautiful colors and layout design

**TUI Keyboard Shortcuts:**

| Shortcut | Function |
|----------|----------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` | Switch to selected configuration |
| `a` | Add new configuration |
| `e` | Edit selected configuration |
| `d` | Delete selected configuration |
| `t` | Toggle app (Claude/Codex) |
| `c` | Switch to Claude |
| `x` | Switch to Codex |
| `r` | Refresh list |
| `q` / `Ctrl+C` | Exit |

In form editing mode:
- `Tab` / `Shift+Tab` / `↑` / `↓` - Switch input focus
- Type directly - Edit the currently focused input field
- `Enter` / `Ctrl+S` - Save and submit
- `ESC` - Cancel and return

### Command-Line Mode

#### Switch Configuration

```bash
ccs <configuration-name>
```

Example:
```bash
ccs 88code
```

Output:
```
✓ Switched to configuration: 88code
  Token: 88_e7...***
  URL: https://www.example.org/api
```

### Add New Configuration

#### Method 1: Interactive Mode

```bash
ccs config add my-config
```

The program will prompt you to enter:
- API Token (hidden input)
- Base URL
- Category (optional)

#### Method 2: Command-line Arguments

```bash
ccs config add my-config \
  --apikey "sk-ant-xxxxx" \
  --base-url "https://api.example.com" \
  --category "custom"
```

Supported category types:
- `official` - Official API
- `cn_official` - Official China region
- `aggregator` - Aggregator service
- `third_party` - Third-party service
- `custom` - Custom (default)

### Delete Configuration

```bash
ccs config delete <configuration-name>
```

Add `--force` or `-f` flag to skip confirmation:

```bash
ccs config delete my-config --force
```

### Codex CLI Configuration Management 🆕

#### Add Codex Configuration

```bash
ccs codex add my-codex \
  --apikey "sk-ant-xxxxx" \
  --base-url "https://api.anthropic.com" \
  --model "claude-3-5-sonnet-20241022"
```

#### List Codex Configurations

```bash
ccs codex list
```

#### Switch Codex Configuration

```bash
ccs codex switch my-codex
```

Output:
```
✓ Switched to Codex configuration: my-codex
  Base URL: https://api.anthropic.com
  API Key: sk-a...***
  Model: claude-3-5-sonnet-20241022

Updated files:
  - C:\Users\username\.codex\config.yaml
  - C:\Users\username\.codex\api.json
```

#### Update Codex Configuration

```bash
ccs codex update my-codex \
  --model "claude-opus-4-20250514" \
  --apikey "sk-new-key"
```

#### Delete Codex Configuration

```bash
ccs codex delete my-codex -f
```

**Codex Configuration Features:**
- 🔄 **Dual File Management** - Automatically maintains `config.yaml` and `api.json`
- ⚡ **Atomic Operations** - Transactional writes with automatic rollback on failure
- 🎯 **Model Support** - Customize the Claude model to use
- 🛡️ **SSOT Pattern** - Fully consistent with Rust backend architecture

### Configuration Backup & Restore 🆕

#### Export Configuration

```bash
# Export to default file (cc-switch-export-<timestamp>.json)
ccs export

# Export to specified file
ccs export --output my-config.json

# Export with filtering by app or provider
ccs export --app claude --pretty
```

#### Import Configuration (with Auto-backup)

```bash
# Import from file (automatically creates backup)
ccs import --from-file my-config.json

# Sample output:
# ✓ Backup created: backup_20251006_143528
# ✓ Configuration imported: Claude Official-1
# Import complete: 1 configuration imported, 0 skipped
```

**Key Features:**
- ✅ **Auto-backup** - Automatically backs up current config to `~/.cc-switch/backups/` before import
- ✅ **Backup Format** - `backup_YYYYMMDD_HHMMSS.json` (consistent with GUI v3.4.0)
- ✅ **Auto-cleanup** - Only keeps the 10 most recent backups, old ones are automatically deleted

#### Backup Management

```bash
# Manually create backup
ccs backup

# List all backups
ccs backup list

# Restore from backup
ccs backup restore backup_20251006_143528

# Sample restore output:
# ✓ Pre-restore backup created: backup_20251006_143639_pre-restore.json
# ✓ Configuration restored from backup: backup_20251006_143528.json
```

**Backup Features:**
- 📦 **Safe Restore** - Automatically backs up current config before restoration
- 🔍 **Format Validation** - Validates backup file integrity before restoration
- 📊 **Detailed Info** - Displays backup time, size, and path

## Configuration File

Configuration file locations:

- **Windows**: `%USERPROFILE%\.cc-switch\config.json`
- **macOS/Linux**: `~/.cc-switch/config.json`

Configuration file format:
```json
{
  "version": 2,
  "claude": {
    "providers": {
      "uuid-xxx": {
        "id": "uuid-xxx",
        "name": "config-name",
        "settingsConfig": {
          "env": {
            "ANTHROPIC_AUTH_TOKEN": "your-api-token",
            "ANTHROPIC_BASE_URL": "api-endpoint"
          }
        },
        "category": "custom",
        "createdAt": 1234567890
      }
    },
    "current": "active-config-id"
  }
}
```

## Compatibility with cc-switch GUI Version

cc-switch-cli is fully compatible with the [cc-switch](https://github.com/YangQing-Lin/cc-switch) GUI version:

- ✅ Shares the same configuration file format
- ✅ Supports identical configuration structure
- ✅ Can be used interchangeably
- ✅ Configuration changes sync in real-time
- ✅ Backup format fully compatible (v0.5.0 aligned with GUI v3.4.0)

You can use both CLI and GUI versions simultaneously as they read and write the same configuration file. Backups created by CLI and GUI are also mutually restorable.

## Security Considerations

1. **File Permissions** - Configuration files default to 600 permissions (owner read/write only)
2. **Token Masking** - API tokens are automatically masked when displayed
3. **Backup Mechanism** - Automatic timestamped backups before import, keeps most recent 10
4. **Input Protection** - API token input is hidden during configuration
5. **Restore Protection** - Automatically backs up current config before restoration

## FAQ

### Q: How to migrate from old version configurations?

A: cc-switch-cli automatically detects and migrates v1 configuration files to v2 format.

### Q: What to do if the configuration file is corrupted?

A: You can restore from the following backups:
1. Use `ccs backup list` to view all automatic backups
2. Use `ccs backup restore <backup-id>` to restore to a specific backup
3. Auto-backups before import are located in `~/.cc-switch/backups/` directory
4. You can also manually restore from `config.json.bak.cli`

### Q: Which Claude API providers are supported?

A: All services compatible with the Anthropic API format are supported, including:
- Official Claude API
- Various relay services
- Local proxy services

### Q: How to verify if a configuration is valid?

A: Basic validation (name, token, URL format) is performed when adding configurations. Actual connection testing occurs during usage.

## Development

### Project Structure

```
cc-switch-cli/
├── main.go                 # Entry point
├── cmd/                    # Command-line interface
│   ├── root.go            # Root command (with TUI integration)
│   ├── ui.go              # TUI subcommand
│   ├── config.go          # Config subcommand
│   ├── add.go             # Add configuration
│   └── delete.go          # Delete configuration
├── internal/              # Internal implementation
│   ├── config/           # Configuration management
│   ├── tui/              # TUI interface (Bubble Tea)
│   ├── i18n/             # Internationalization support
│   └── utils/            # Utility functions
└── go.mod                # Dependency management
```

### Technology Stack

- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) - Command-line interface
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal user interface
- **TUI Components**: [Bubbles](https://github.com/charmbracelet/bubbles) - Reusable components
- **Style Beautification**: [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

### Building the Project

```bash
# Build for current platform
go build -o cc-switch

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o cc-switch.exe

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o cc-switch-darwin

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o cc-switch-linux
```

### Running Tests

This project includes comprehensive unit tests and integration tests:

```bash
# Run all tests
go test ./...

# Run unit tests with coverage
go test -cover ./internal/...

# Run integration tests
go test -v ./test/integration/...

# Use test scripts
./test.bat           # Windows
./test.sh            # Linux/macOS

# Generate coverage report
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

#### Test Coverage

- **internal/utils**: 69.7% - Atomic file operations, JSON I/O
- **internal/settings**: 82.4% - Settings management, language switching
- **internal/i18n**: 60.0% - Internationalization support (EN/ZH)
- **internal/config**: 32.1% - Provider CRUD, configuration management

#### Integration Tests

Integration tests verify multiple components working together:

- ✅ Provider CRUD operations
- ✅ Configuration persistence (simulated restart)
- ✅ Multi-app support (Claude/Codex)
- ✅ Configuration file structure validation
- ✅ Concurrent access protection
- ✅ Data integrity verification

See [docs/testing.md](docs/testing.md) for detailed testing documentation.

## License

MIT License

## Contributing

Issues and Pull Requests are welcome!

## Related Projects

- [cc-switch](https://github.com/YangQing-Lin/cc-switch) - GUI version with graphical interface for configuration management