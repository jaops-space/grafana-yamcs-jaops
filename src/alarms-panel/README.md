# Alarms Panel

A Grafana panel plugin for monitoring and managing Yamcs alarms in real-time via WebSocket streaming. This panel provides full-featured alarm management matching the functionality of Yamcs Web.

## Features

### Core Features
- **Real-time Monitoring**: Live table of active alarms with automatic WebSocket updates
- **Global Alarm Status**: Dashboard-wide alarm status summary showing unacknowledged, acknowledged, and shelved alarm counts with severity levels
- **Alarm Actions**: Acknowledge, Clear, Shelve, and Unshelve alarms with optional comments and confirmation dialogs
- **Severity Indicators**: Color-coded levels: Watch (blue), Warning/Distress (orange), Critical/Severe (red)
- **Status Display**: Clear text-based status showing alarm state (Triggered, Acknowledged, Shelved, OK)
- **Trip Value Column**: Displays the parameter value that triggered the alarm, positioned between Alarm Type and Live Value
- **Expandable Details**: View full parameter path, trip/live values, violation counts, acknowledgement history, shelve information, and action comments
- **Action Audit Trail**: All alarm actions (acknowledge, shelve, clear) include who performed the action, when, and optional comments

### Table Columns

The alarm table displays the following columns (from left to right):

| Column | Description |
|--------|-------------|
| **Severity** | Visual indicator (icon + text) showing alarm severity level |
| **Alarm time** | Timestamp when the alarm was triggered |
| **Alarm name** | Full qualified parameter name (namespace/parameter) |
| **Alarm type** | Type of alarm (e.g., PARAMETER, EVENT) |
| **Trip value** | The parameter value that triggered the alarm |
| **Live value** | Current value of the parameter |
| **Status** | Current alarm state (Triggered, Acknowledged, Shelved, OK) |
| **Actions** | Quick action buttons for alarm management |

## Usage

1. Add a new panel -> select the **JAOPS Yamcs** datasource
2. Set **Query Type** = `Alarms` and select your endpoint
3. Active alarms appear automatically via WebSocket streaming
4. View the Global Alarm Status bar above the table for an overview of all alarms
5. Use the action buttons in each row to manage alarms:
   - **Acknowledge** (✓): Mark an alarm as seen
   - **Clear** (✗): Remove an acknowledged alarm from the active list
   - **Shelve** (🕘): Temporarily hide an alarm
   - **Unshelve** (👁): Restore a shelved alarm to visibility
6. Click the expand arrow (▶) on any row to see detailed information including comments from previous actions

## Global Alarm Status

The status bar at the top of the panel displays:

- **Unacknowledged Alarms**: Count of active alarms that haven't been acknowledged (red)
- **Acknowledged Alarms**: Count of acknowledged alarms that are still active (blue)
- **Shelved Alarms**: Count of alarms that have been temporarily hidden (gray)
- **No Active Alarms**: Displayed when all alarm counts are zero (green)

## Panel Options

| Option | Description |
|---|---|
| Visible Columns | Select which fields to display (default includes all columns) |
| Show Details on Expand | Toggle expandable row details with full alarm information |
| Enable Pagination | Toggle pagination on/off |
| Page Size | Rows per page (when pagination enabled) |

## Alarm Actions

### Acknowledge
Mark an alarm as seen by an operator. This indicates that someone is aware of the issue.

### Clear
Remove an acknowledged alarm from the active alarms list. Only acknowledged alarms can be cleared.

### Shelve
Temporarily hide an alarm from the active display. Useful for known issues being investigated.

### Unshelve
Restore a shelved alarm to make it visible again in the active alarms list.

## Alarm Status

The **Status** column displays the current state of each alarm:

| Status | Icon | Color | Meaning |
|--------|------|-------|---------|
| **Triggered** | ⚠️ | Red | Alarm is active and not yet acknowledged |
| **Acknowledged** | ✓ | Blue | Alarm has been acknowledged by an operator |
| **Shelved** | 🕘 | Gray | Alarm is temporarily hidden |
| **OK** | ✓ | Green | Parameter is within limits (alarm cleared automatically) |


## Testing

### Automated

Run the test script to validate Yamcs alarm APIs and Grafana resource endpoints:

```bash
cd src/alarms-panel
chmod +x test-alarm-logic.sh
./test-alarm-logic.sh
```

> Update `DATASOURCE_UID` and `ENDPOINT_ID` in the script to match your setup.

### End-to-End (Manual)

**Prerequisites:** Yamcs simulator + Grafana running (e.g. `docker-compose up -d --build`).  
The default Yamcs quickstart simulator (`yamcs/example-simulation`) has alarms pre-configured in its MDB and generates out-of-limit values automatically (~30s after startup).

1. Start the simulator: `docker run -d -p 8091:8090 yamcs/example-simulation`
2. Open Grafana -> create a panel with **Query Type** = `Alarms`
3. Wait for alarms to appear, then test Acknowledge / Clear / Shelve buttons
