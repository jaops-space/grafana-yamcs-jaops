# Alarms Panel

A Grafana panel plugin for monitoring and managing Yamcs alarms in real-time via WebSocket streaming.

## Features

- **Real-time Monitoring**: Live table of active alarms with automatic WebSocket updates
- **Alarm Actions**: Acknowledge, Clear, and Shelve alarms with confirmation dialogs
- **Severity Indicators**: Color-coded levels: Watch (blue), Warning/Distress (orange), Critical/Severe (red)
- **Expandable Details**: View trigger values, acknowledgement history, and sample counts
- **Configurable Columns & Pagination**: Choose visible fields and enable pagination for large lists

## Usage

1. Add a new panel → select the **JAOPS Yamcs** datasource
2. Set **Query Type** = `Alarms` and select your endpoint
3. Active alarms appear automatically via WebSocket streaming
4. Use the action buttons in each row to **Acknowledge** (✓), **Clear** (✗), or **Shelve** (🕘) alarms

## Panel Options

| Option | Description |
|---|---|
| Visible Columns | Select which fields to display |
| Show Details on Expand | Toggle expandable row details |
| Enable Pagination | Toggle pagination on/off |
| Page Size | Rows per page (when pagination enabled) |

## Alarm Actions

- **Acknowledge**: Mark an alarm as seen by an operator
- **Clear**: Remove an acknowledged alarm once the parameter is back within limits
- **Shelve**: Temporarily hide an alarm (default: 1 hour) for known issues

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
2. Open Grafana → create a panel with **Query Type** = `Alarms`
3. Wait for alarms to appear, then test Acknowledge / Clear / Shelve buttons
