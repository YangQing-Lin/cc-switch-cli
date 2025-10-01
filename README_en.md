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

## Installation

### Build from Source

Requires Go 1.25.1 or higher:

```bash
git clone https://github.com/YangQing-Lin/cc-switch-cli.git
cd cc-switch-cli
go build -o cc-switch
```

### Download Pre-built Binaries

Download the appropriate pre-compiled binary for your operating system from the [Releases](https://github.com/YangQing-Lin/cc-switch-cli/releases) page.

## Usage

### Interactive TUI Interface (Recommended)

Launch the interactive terminal user interface:

```bash
cc-switch
# Or explicitly specify
cc-switch ui
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
| `↑` / `↓` | Move cursor up/down (select configuration) |
| `Enter` | Switch to selected configuration |
| `a` | Add new configuration |
| `e` | Edit selected configuration |
| `d` | Delete selected configuration |
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
cc-switch <configuration-name>
```

Example:
```bash
cc-switch 88code
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
cc-switch config add my-config
```

The program will prompt you to enter:
- API Token (hidden input)
- Base URL
- Category (optional)

#### Method 2: Command-line Arguments

```bash
cc-switch config add my-config \
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
cc-switch config delete <configuration-name>
```

Add `--force` or `-f` flag to skip confirmation:

```bash
cc-switch config delete my-config --force
```

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

You can use both CLI and GUI versions simultaneously as they read and write the same configuration file.

## Security Considerations

1. **File Permissions** - Configuration files default to 600 permissions (owner read/write only)
2. **Token Masking** - API tokens are automatically masked when displayed
3. **Backup Mechanism** - Automatic `.bak` backup file created before each save
4. **Input Protection** - API token input is hidden during configuration

## FAQ

### Q: How to migrate from old version configurations?

A: cc-switch-cli automatically detects and migrates v1 configuration files to v2 format.

### Q: What to do if the configuration file is corrupted?

A: You can restore from the automatically generated `config.json.bak` backup file.

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
- **internal/vscode**: 25.0% - VS Code/Cursor integration
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