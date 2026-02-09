# JAOPS Links Panel

A Grafana panel plugin for managing Yamcs data links directly from your dashboards.

## Features

- **View all Yamcs links** - Display link status, type, class, and data counters
- **Enable/Disable links** - Toggle links on/off with a single click
- **Reset counters** - Reset data in/out counters for individual links
- **Auto-refresh** - Configurable automatic refresh interval
- **Filtering** - Filter links by name using regex patterns
- **Variable support** - Use dashboard variables for dynamic endpoint selection

## Panel Options

| Option | Description | Default |
|--------|-------------|---------|
| Auto-refresh interval | Automatically refresh link status (0 = manual only) | 5 seconds |
| Show details | Display detailed link information (type, class, data counts) | true |
| Filter by name | Regex pattern to filter links by name | empty |

## Usage

1. Add the "JAOPS Links Panel" to your dashboard
2. Configure the query:
   - Select the Yamcs datasource
   - Choose an endpoint from the dropdown OR check "As variable" to use a dashboard variable
   - Set Query Type to "Links"
3. Links will be displayed with their current status
4. Use the Enable/Disable buttons to toggle link state
5. Use the Reset button to clear data counters
6. Use the refresh button or auto-refresh for live updates

## Variable Support

To use a dashboard variable for the endpoint:
1. Check the "As variable" checkbox in the query editor
2. Enter the variable name (e.g., `$endpoint`)
3. The panel will dynamically update when the variable changes

## Status Colors

- **Green** - Link is OK and operational
- **Orange** - Link is disabled
- **Red** - Link has failed or is unavailable
- **Blue** - Link status is unknown