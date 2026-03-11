# Alarms Panel
A Grafana panel plugin for monitoring and managing Yamcs alarms in real-time via WebSocket streaming. This panel provides full-featured alarm management matching the functionality of Yamcs Web.
## Features
### Core Features
- **Real-time Monitoring**: Live table of active alarms with automatic WebSocket updates
- **Global Alarm Status**: Dashboard-wide alarm status summary showing unacknowledged, acknowledged, and shelved alarm counts (always visible, including zero counts)
- **Dual Alarm Type Support**: Full support for both **Parameter Alarms** (out-of-limit conditions) and **Event Alarms** (severity-based event triggers)
- **Alarm Actions**: Acknowledge, Clear, Shelve (with duration selector: 15 min to unlimited), and Unshelve alarms with optional comments and confirmation dialogs
- **5 Distinct Severity Levels**: Color-coded solid circles for Watch (light blue), Warning (orange), Distress (dark orange), Critical (red), Severe (dark red) - no information loss
- **Precise Duration Display**: "Alarm time" column shows exact duration (e.g., "56 minutes ago", "1h 10 minutes ago") for quick assessment
- **Status Display**: Clear state indicators in first column showing alarm status (Triggered, Acknowledged, Shelved, Cleared, OK)
- **Optimized Column Order**: Matches Yamcs Web layout (State, Severity, Alarm time, Trigger Timestamp, Alarm name, Type, Trigger value, Live value, Actions)
- **Expandable Details**: View full parameter path or event source, trigger/most severe/live values or event messages, violation counts, acknowledgement history, shelve information, and action comments. For parameter alarms, also includes detailed ParameterValue objects (eng/raw values, acquisition/generation times, monitoring results) and ParameterInfo metadata.
- **Action Audit Trail**: All alarm actions (acknowledge, shelve, clear) include who performed the action, when, and optional comments


## Usage
1. Add a new panel -> select the **JAOPS Yamcs** datasource
2. Set **Query Type** = `Alarms` and select your endpoint
3. Active alarms appear automatically via WebSocket streaming
4. View the Global Alarm Status bar above the table for a complete overview (all categories always visible)
5. Use the action buttons in each row to manage alarms:
   - **Acknowledge** (✓): Mark an alarm as seen
   - **Clear** (✗): Remove an acknowledged alarm from the active list
   - **Shelve** (🕘): Temporarily hide an alarm (select duration: 15 min, 30 min, 1h, 2h, 1 day, or unlimited)
   - **Unshelve** (👁): Restore a shelved alarm to visibility
6. Click the expand arrow (▶) on any row to see detailed information including full parameter path and action history
## Panel Options
| Option | Description |
|--------|-------------|
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

## Using with Grafana Alerts

This panel displays **YAMCS-native alarms** from your Mission Database (MDB). You can also use **[Grafana Alerts](https://grafana.com/docs/grafana/latest/alerting/)** in parallel for custom notifications.

**YAMCS Alarms Panel** (this panel):
- View official YAMCS alarms with full operator workflow (acknowledge/shelve/clear)
- Primary interface for mission control

**Grafana Alerts** (native Grafana feature):
- Create custom alert rules on any YAMCS telemetry parameter
- Set up notifications (email, Slack, PagerDuty, etc.)
- Define thresholds not in your YAMCS MDB

**Example**: Create a Grafana Alert that sends a Slack notification when `BatteryVoltage1 < 50V for 5 minutes`, while still viewing all YAMCS MDB alarms in this panel.
## Testing
### Automated Testing
Run the comprehensive test script to validate all alarm panel features:
```bash
cd src/alarms-panel
chmod +x test-alarm-logic.sh
./test-alarm-logic.sh
```
#### What the Test Script Validates

- Yamcs simulator alarm APIs (acknowledge, clear, shelve, unshelve)
- Grafana datasource alarm endpoints
- Trigger value extraction from parameter alarms
- Event alarm generation and processing
- Event alarm data structure (eventDetail with trigger/current events)
- Event alarm actions (acknowledge, shelve, clear, unshelve)
- Global alarm status calculation
- Comment persistence for all alarm actions
- Alarm state transitions (Triggered → Acknowledged → Cleared/Shelved)
- Parameter vs Event alarm type comparison
- Alarm cache persistence (alarms remain visible over time)
- Alarm sorting stability (consistent ordering across requests)
### Manual Testing - Parameter Alarms
**Prerequisites:** Yamcs simulator + Grafana running.

The default Yamcs quickstart simulator (`yamcs/example-simulation`) has parameter alarms pre-configured in its MDB and generates out-of-limit values automatically (~30s after startup).

```bash
docker run -d -p 8090:8090 yamcs/example-simulation
```

#### Test Procedure:

1. **Start the simulator** (use either option above)
2. **Open Grafana** -> Create a panel with **Query Type** = `Alarms`
3. **Wait for parameter alarms** to appear (typically `BatteryVoltage1`, `BatteryVoltage2`)
4. **Verify parameter alarm display**:
   - Alarm type shows `PARAMETER`
   - Trigger value shows the numeric value that triggered the alarm (e.g., "57")
   - Live value shows the current parameter value
   - Global Alarm Status shows counts above the table
   - Alarms remain in consistent order
5. **Test Acknowledge**:
   - Click the ✓ button on an unacknowledged alarm
   - Add a comment (e.g., "Investigating issue")
   - Confirm and verify status changes to "Acknowledged"
   - Expand the row and verify the comment is displayed
6. **Test Shelve**:
   - Click the 🕘 button on an alarm
   - Add a comment (e.g., "Known issue, will fix in next release")
   - Confirm and verify status changes to "Shelved"
   - Expand the row and verify shelve information (time, expiration, comment)
7. **Test Unshelve**:
   - Click the 👁 button on a shelved alarm
   - Verify the alarm returns to its previous state
8. **Test Clear**:
   - On an acknowledged alarm, click the ✗ button
   - Add a comment (e.g., "Issue resolved")
   - Confirm and verify the alarm is removed from the list
### Manual Testing - Event Alarms
Event alarms are automatically generated when events with severity > INFO are received.
#### Test Procedure:
1. **Ensure the simulator is running** at `http://localhost:8090`
2. **Generate a WARNING event alarm**:
   ```bash
   curl -X POST "http://localhost:8090/api/archive/simulator/events" \
     -H "Content-Type: application/json" \
     -d '{"message": "This is a test event alarm", "source": "TestSource", "type": "TestAlarm", "severity": "WARNING"}'
   ```
3. **Generate a CRITICAL event alarm**:
   ```bash
   curl -X POST "http://localhost:8090/api/archive/simulator/events" \
     -H "Content-Type: application/json" \
     -d '{"message": "Critical system failure detected", "source": "SystemMonitor", "type": "SystemFailure", "severity": "CRITICAL"}'
   ```
4. **Verify event alarm display** in the Grafana panel:
   -  Alarm type shows `EVENT`
   - Alarm name shows event source and type (e.g., "/yamcs/event/TestSource/TestAlarm")
   - Trigger value shows trigger event message with severity (e.g., "WARNING: This is a test event alarm")
   - Live value shows current event message with severity
   - Both parameter and event alarms visible simultaneously
   - Alarms remain in consistent order (no jumping)
5. **Test alarm actions**: Event alarms support all the same actions as parameter alarms:
   - Acknowledge with comment
   - Shelve with comment and duration
   - Unshelve
   - Clear (after acknowledging)
6. **Expand row details** to verify:
   - Shows "Event Source/Type" instead of "Full Parameter Path"
   - Shows "Trigger Event" and "Current Event" labels
   - Displays event messages with severity
