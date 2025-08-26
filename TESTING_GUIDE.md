# Testing Guide for Yamcs Plugin

This guide provides instructions for testing the Yamcs Grafana plugin functionality.

## Prerequisites

1. **Yamcs Server**: You need a running Yamcs server instance
   - Setup the quickstart from: https://github.com/jaops-space/yamcs-quickstart
   
2. **Grafana Instance**: Grafana 10.4.0 or later

## Quick Setup for Testing

### 1. Install the Plugin

1. Download the plugin ZIP from the release
2. Extract to your Grafana plugins directory
3. Restart Grafana
4. Enable the plugin in Configuration > Plugins

### 2. Configure Yamcs Connection

1. Go to Configuration > Data Sources
2. Add new Yamcs data source
3. Configure connection settings:
   - **URL**: Your Yamcs server URL (e.g., `http://localhost:8090`)
   - **Instance**: Yamcs instance name
   - **Credentials**: If authentication is required

### 3. Test Basic Functionality

#### Telemetry Display
1. Create a new dashboard
2. Add a graph panel
3. Select Yamcs as data source
4. Query telemetry parameters
5. Verify data display

#### Commanding
1. Navigate to the Commanding page in the plugin
2. Configure command buttons
3. Test command execution
4. Verify command history

#### Image Panels
1. Add an Image Panel
2. Configure with static image
3. Overlay telemetry data
4. Test real-time updates

## Sample Data

The yamcs-quickstart repository includes:
- Sample Yamcs configuration
- Test telemetry data and simulators (for telemetry, images, etc)

This grafana-yamcs-jaops repository includes:
- Example dashboards in `provisioning/`

## Troubleshooting

### Common Issues
1. **Connection Failed**: Check Yamcs server is running and accessible
2. **No Data**: Verify Yamcs instance name and parameter names
3. **Commands Not Working**: Check command definitions and permissions

### Logs
- Check Grafana logs for plugin errors
- Yamcs server logs for connection issues
- Browser console for frontend errors

## Expected Test Results

- ✅ Plugin loads without errors
- ✅ Data source connects to Yamcs
- ✅ Telemetry data displays correctly
- ✅ Commands execute successfully
- ✅ Image panels overlay data properly
- ✅ Real-time updates work smoothly
