# Commanding Panel

Interactive interface for sending YAMCS telecommands directly from Grafana dashboards.

## Features

- Execute YAMCS commands with automatic form validation
- Customize button appearance (icons, colors, shapes, backgrounds)
- Dual button mode for on/off command pairs
- Set Grafana variables on command execution
- Custom SVG shapes support

## Configuration

### Button Settings
- **Label**: Display text
- **Icon**: Choose from Grafana icons
- **Color/Text Color**: Customize appearance
- **Size**: small, medium, large
- **Shape**: circle, square, rounded, or custom SVG

### Command Settings
- **Command**: Select from YAMCS commands
- **Arguments**: Auto-generated form with validation (number ranges, enums, etc.)
- **Comment**: Add notes about the command

### Dual Button Mode
Enable to create on/off button pairs. Configure both commands separately with independent styling.

### Variable Integration
Enable to update Grafana variables on command execution:
- Select target variable
- Choose mode: `change` (replace), `add` (append), or `multiply`

## Usage

1. Click a button to execute the command
2. For commands with arguments, fill in the form and send
3. Visual feedback indicates success/error status
4. Related variables automatically refresh

## Limitations

- One command execution at a time (queued)
- State stored in browser storage (resets on refresh)
- See Command History Panel for execution logs