# GEMINI.md

## Project Overview

`cc-switch-cli` is a command-line tool written in Go for managing multiple Claude API configurations. It provides a terminal user interface (TUI) for easy switching between different API keys and endpoints. The tool is designed to be cross-platform and supports interactive and non-interactive modes. It also includes features for configuration backup, restore, and portable execution.

The project uses the Cobra library for its command-line interface and Bubble Tea for the TUI.

## Building and Running

### Building

To build the project, you need Go 1.25.1 or higher.

```bash
go build -o ccs
```

This will create an executable named `ccs` in the project directory.

### Running

The main command to run the application is `ccs`. Running `ccs` without any arguments will start the interactive TUI.

Here are some common commands:

*   `ccs ui`: Explicitly start the TUI.
*   `ccs config show`: List all configurations.
*   `ccs <config-name>`: Switch to a specific configuration.
*   `ccs config add <config-name>`: Add a new configuration.
*   `ccs config delete <config-name>`: Delete a configuration.

### Testing

To run the tests, use the following command:

```bash
go test ./...
```

## Development Conventions

*   **CLI:** The command-line interface is built using the Cobra library. Commands are organized in the `cmd/` directory.
*   **TUI:** The terminal user interface is built with Bubble Tea and its components are located in the `internal/tui/` directory.
*   **Configuration:** Configuration management logic is in the `internal/config/` directory.
*   **Dependencies:** Project dependencies are managed with Go Modules (`go.mod` and `go.sum`).
*   **Internationalization:** The project includes support for internationalization (i18n), with localization files in `internal/i18n/`.
*   **Testing:** The project has both unit and integration tests. Test files are located alongside the source files with a `_test.go` suffix, and integration tests are in the `test/integration/` directory.
