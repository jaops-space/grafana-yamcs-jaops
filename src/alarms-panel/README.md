# Alarms Panel

A Grafana panel plugin for monitoring and managing Yamcs alarms in real-time.

## Features

- **Real-time Alarm Monitoring**: View active alarms from a Yamcs processor with automatic updates
- **Alarm Actions**: Acknowledge, clear and shelve alarms directly from the panel
- **Severity Indicators**: Color-coded severity levels (Watch, Warning, Distress, Critical, Severe)
- **Detailed View**: Expandable rows showing alarm details including trigger values and acknowledgement history
- **Configurable Columns**: Choose which columns to display in the table
- **Pagination Support**: Optional pagination for large alarm lists

## Configuration

### Panel Options

- **Visible Columns**: Select which fields to display in the table
- **Show Details on Expand**: Enable/disable expandable row details
- **Enable Pagination**: Toggle pagination on/off
- **Page Size**: Number of alarms per page (when pagination is enabled)

## Usage

1. Add a new panel to your dashboard
2. Select the JAOPS Yamcs datasource
3. Choose "Alarms" as the query type
4. Select your endpoint
5. Configure panel options as needed

## Alarm Actions

### Acknowledge
Mark an alarm as acknowledged. This indicates that an operator is aware of the alarm condition.

### Clear
Remove an acknowledged alarm when the parameter has returned to normal limits.

### Shelve
Temporarily hide an alarm from the active list. Useful for known issues that don't require immediate attention.
