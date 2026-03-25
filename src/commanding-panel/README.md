# Commanding Panel

Interactive interface for sending Yamcs telecommands directly from Grafana dashboards.

## Features

- Execute Yamcs commands with automatic form validation based on Yamcs Mission Data Base (e.g command arguments)
- Customize button appearance (icons, colors, shapes, backgrounds)
- Dual button mode for command pairs (e.g ON/OFF)
- Custom SVG shapes support

## Configuration

### Button Settings

- **Label**: Display text
- **Icon**: Choose from Grafana icons
- **Color/Text Color**: Customize appearance
- **Size**: small, medium, large
- **Shape**: circle, square, rounded, or custom SVG

### Command Settings

- **Command**: Select from Yamcs commands
- **Arguments**: Auto-generated form with validation (number ranges, enums, etc.)
- **Comment**: Add notes about the command

### Dual Button Mode

Enable to create on/off button pairs. Configure both commands separately with independent styling.

## Usage

1. Click a button to execute the command
2. For commands with arguments, fill in the form and send
3. Visual feedback indicates success/error status
4. See Command History Panel for execution logs

## Limitations

- One command execution at a time (no support for command stacks yet!)
- State of the dual button mode is stored in browser storage (resets on refresh)
