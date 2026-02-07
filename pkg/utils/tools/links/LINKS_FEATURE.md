# Yamcs Link Management Feature

> **Feature Summary**: Control Yamcs data links directly from Grafana - enable/disable links, reset counters, and execute link actions through REST API endpoints. Includes automated testing tools and browser console integration.

## What This Feature Does

Manage Yamcs links programmatically from Grafana without accessing the Yamcs server directly:

- **📋 List Links** - View all configured links with status, counters, and available actions
- **🔍 Get Link Details** - Retrieve specific link information including type, state, and metrics
- **🟢 Enable/Disable Links** - Control link state (useful for SLE links, TC/TM links, etc.)
- **🔄 Reset Counters** - Clear data in/out statistics for links
- **⚡ Run Actions** - Execute link-specific operations defined by the link implementation

**Related**: [Issue #7](https://github.com/jaops-space/grafana-yamcs-jaops/issues/7) - Feature request for managing Yamcs links and alarms from Grafana.

## Quick Start

**Prerequisites**: Grafana with grafana-yamcs-jaops plugin, configured Yamcs datasource, and accessible Yamcs server with links.

### Test in 3 Ways

1. **Automated Python Script** (Recommended)
   ```bash
   python3 test_link_management.py
   # Interactive prompts guide you through testing
   ```

2. **Browser Console** (Quick)
   - Open Grafana → Press F12 → Console tab
   - Run provided JavaScript snippets to test APIs

3. **curl Commands** (Manual)
   ```bash
   curl -s "http://localhost:3000/api/datasources/uid/YOUR_UID/resources/endpoint/YOUR_ENDPOINT/links" | jq
   ```

Detailed instructions for each method below.

## API Reference

All endpoints use: `/api/datasources/uid/{datasourceUID}/resources/endpoint/{endpointID}/...`

| Operation | Method | Path | Description |
|-----------|--------|------|-------------|
| **List Links** | GET | `/links` | Get all links with status and actions |
| **Get Link** | GET | `/links/{linkName}` | Get specific link details |
| **Enable** | POST | `/links/{linkName}/enable` | Enable a disabled link |
| **Disable** | POST | `/links/{linkName}/disable` | Disable an active link |
| **Reset** | POST | `/links/{linkName}/reset` | Reset data counters to zero |
| **Action** | POST | `/links/{linkName}/action/{actionID}` | Execute link action |

<details>
<summary><b>Example Response (Click to expand)</b></summary>

```json
[
  {
    "instance": "simulator",
    "name": "udp-in",
    "type": "org.yamcs.tctm.UdpTmDataLink",
    "disabled": false,
    "status": "OK",
    "dataInCount": 12345,
    "dataOutCount": 0,
    "detailedStatus": "Receiving data",
    "actions": [
      {
        "id": "resetCounters",
        "label": "Reset Counters",
        "style": "PUSH_BUTTON",
        "enabled": true
      }
    ]
  }
]
```
</details>

---

## Testing Methods

### Method 1: Automated Python Test Script ⭐

**Recommended for comprehensive testing**

The `test_link_management.py` script provides automated testing with colored output and interactive prompts.

```bash
# Interactive mode - prompts for datasource UID and endpoint
python3 test_link_management.py

# Specify everything on command line
python3 test_link_management.py --datasource-uid YOUR_UID --endpoint YOUR_ENDPOINT

# List available endpoints first
python3 test_link_management.py --datasource-uid YOUR_UID --list-endpoints
```

**What it tests**: All operations (list, get, enable/disable toggle, reset counters, run actions)

<details>
<summary>Additional options</summary>

<details>
<summary>Additional options</summary>

```bash
# With authentication
python3 test_link_management.py --username admin --password admin

# Skip action testing
python3 test_link_management.py --skip-actions

# Custom Grafana URL
python3 test_link_management.py --grafana-url http://grafana.example.com:3000
```
</details>

**Expected output**: Color-coded test results showing ✓ passed or ✗ failed for each operation.

---

### Method 2: Browser Developer Console

**Best for quick testing while using Grafana**

1. Open Grafana → Press **F12** → **Console** tab
2. Find your datasource UID:
   ```javascript
   // Option 1: Check URL at Connections → Data sources → Your Yamcs datasource
   // Option 2: Run this:
   fetch('/api/datasources').then(r => r.json()).then(datasources => {
     const yamcs = datasources.find(ds => ds.type === 'jaops-yamcs-app-datasource');
     console.log('Yamcs UID:', yamcs?.uid);
   });
   ```

3. Test the API (replace `YOUR_UID`, `YOUR_ENDPOINT`, `LINK_NAME`):
   ```javascript
   const uid = 'YOUR_UID', endpoint = 'YOUR_ENDPOINT', link = 'udp-in';
   const base = `/api/datasources/uid/${uid}/resources`;
   
   // List links
   fetch(`${base}/endpoint/${endpoint}/links`).then(r => r.json()).then(console.log);
   
   // Disable link
   fetch(`${base}/endpoint/${endpoint}/links/${link}/disable`, {method: 'POST'})
     .then(r => r.json()).then(console.log);
   ```

<details>
<summary>Complete test suite for console</summary>

```javascript
async function testLinkManagement(datasourceUid, endpointId) {
    console.log('Starting Link Management Tests...\n');
    
    const baseUrl = `/api/datasources/uid/${datasourceUid}/resources`;
    
    async function request(path, method = 'GET', body = null) {
        const response = await fetch(`${baseUrl}/${path}`, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: body ? JSON.stringify(body) : null
        });
        return response.json();
    }
    
    try {
        // Test 1: List links
        console.log('Test 1: Listing links...');
        const links = await request(`endpoint/${endpointId}/links`);
        console.log(`✅ Found ${links.length} link(s):`, links.map(l => l.name));
        
        if (links.length === 0) {
            console.warn('⚠️ No links found. Tests stopped.');
            return;
        }
        
        // Use first link for further tests
        const testLink = links[0];
        const linkName = testLink.name;
        console.log(`\n Using link "${linkName}" for tests\n`);
        
        // Test 2: Get specific link
        console.log(' Test 2: Getting link details...');
        const linkDetails = await request(`endpoint/${endpointId}/links/${linkName}`);
        console.log('✅ Link details:', {
            name: linkDetails.name,
            type: linkDetails.type,
            status: linkDetails.status,
            disabled: linkDetails.disabled,
            dataInCount: linkDetails.dataInCount,
            dataOutCount: linkDetails.dataOutCount
        });
        
        // Test 3: Disable link
        console.log('\n🔴 Test 3: Disabling link...');
        const disabledLink = await request(`endpoint/${endpointId}/links/${linkName}/disable`, 'POST');
        console.log(`✅ Link disabled:`, disabledLink.disabled === true ? 'Yes' : 'No');
        
        // Wait a moment
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // Test 4: Enable link
        console.log('\n🟢 Test 4: Enabling link...');
        const enabledLink = await request(`endpoint/${endpointId}/links/${linkName}/enable`, 'POST');
        console.log(`✅ Link enabled:`, enabledLink.disabled === false ? 'Yes' : 'No');
        
        // Test 5: Reset counters
        console.log('\n Test 5: Resetting counters...');
        const resetLink = await request(`endpoint/${endpointId}/links/${linkName}/reset`, 'POST');
        console.log(`✅ Counters reset - In: ${resetLink.dataInCount}, Out: ${resetLink.dataOutCount}`);
        
        // Test 6: Run actions (if available)
        if (testLink.actions && testLink.actions.length > 0) {
            const action = testLink.actions.find(a => a.enabled);
            if (action) {
                console.log(`\n⚡ Test 6: Running action "${action.id}"...`);
                const actionResult = await request(
                    `endpoint/${endpointId}/links/${linkName}/action/${action.id}`, 
                    'POST', 
                    { message: {} }
                );
                console.log('✅ Action completed:', actionResult);
            }
        }
        
        console.log('\n✨ All tests completed successfully!');
        
    } catch (error) {
        console.error('❌ Test failed:', error);
    }
}

// Run the tests - REPLACE WITH YOUR VALUES
testLinkManagement('YOUR_DATASOURCE_UID', 'YOUR_ENDPOINT_ID');
```

#### Step 5: Monitor Network Activity

1. Go to the **Network** tab in Developer Tools
2. Filter by "resources" to see only API calls
3. Run any of the console commands above
4. Click on a request to see:
   - Request URL and method
   - Request headers and payload
   - Response status and data
   - Timing information

#### Interactive Link Management (Advanced)

Create a simple UI in the browser console to manage links interactively:

```javascript
// Create a link management panel
(async function() {
    const datasourceUid = 'YOUR_DATASOURCE_UID';
    const endpointId = 'YOUR_ENDPOINT_ID';
    const baseUrl = `/api/datasources/uid/${datasourceUid}/resources`;
    
    async function request(path, method = 'GET', body = null) {
        const response = await fetch(`${baseUrl}/${path}`, {
            method,
            headers: { 'Content-Type': 'application/json' },
            body: body ? JSON.stringify(body) : null
        });
        return response.json();
    }
    
    // Fetch links
    const links = await request(`endpoint/${endpointId}/links`);
    
    console.log('\n🔗 Link Management Panel\n');
    console.log('Available commands:');
    console.log('');
    
    links.forEach((link, index) => {
        const status = link.disabled ? '🔴 DISABLED' : '🟢 ENABLED';
        console.log(`${index + 1}. ${link.name} - ${status}`);
        console.log(`   Status: ${link.status}`);
        console.log(`   Data In: ${link.dataInCount}, Data Out: ${link.dataOutCount}`);
        
        // Create helper functions for each link
        window[`enable_${link.name.replace(/[^a-zA-Z0-9]/g, '_')}`] = async () => {
            const result = await request(`endpoint/${endpointId}/links/${link.name}/enable`, 'POST');
            console.log(`✅ Enabled ${link.name}:`, result.disabled === false);
            return result;
        };
        
        window[`disable_${link.name.replace(/[^a-zA-Z0-9]/g, '_')}`] = async () => {
            const result = await request(`endpoint/${endpointId}/links/${link.name}/disable`, 'POST');
            console.log(`🔴 Disabled ${link.name}:`, result.disabled === true);
            return result;
        };
        
        window[`reset_${link.name.replace(/[^a-zA-Z0-9]/g, '_')}`] = async () => {
            const result = await request(`endpoint/${endpointId}/links/${link.name}/reset`, 'POST');
            console.log(`🔄 Reset ${link.name} counters:`, result);
            return result;
        };
        
        console.log(`   enable_${link.name.replace(/[^a-zA-Z0-9]/g, '_')}() - Enable this link`);
        console.log(`   disable_${link.name.replace(/[^a-zA-Z0-9]/g, '_')}() - Disable this link`);
        console.log(`   reset_${link.name.replace(/[^a-zA-Z0-9]/g, '_')}() - Reset counters`);
        console.log('');
    });
    
    console.log('Example: enable_udp_in()');
})();
```

Now you can call functions like `enable_udp_in()` or `disable_tcp_tm()` directly from the console!

### Method 4: Integration from Grafana Panels

You can integrate link management into your Grafana panels or apps using the datasource API:

```typescript
import { getBackendSrv } from '@grafana/runtime';

// In your component or service
const datasourceUid = 'your-datasource-uid';
const endpointId = 'your-endpoint';
const linkName = 'udp-in';

// List links
async function listLinks() {
    const response = await getBackendSrv().fetch({
        url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${endpointId}/links`,
        method: 'GET',
    });
    return response.data;
}

// Enable a link
async function enableLink() {
    const response = await getBackendSrv().fetch({
        url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${endpointId}/links/${linkName}/enable`,
        method: 'POST',
    });
    return response.data;
}

// Disable a link
async function disableLink() {
    const response = await getBackendSrv().fetch({
        url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${endpointId}/links/${linkName}/disable`,
        method: 'POST',
    });
    return response.data;
}

// Run link action
async function runLinkAction(actionId: string, message?: any) {
    const response = await getBackendSrv().fetch({
        url: `/api/datasources/uid/${datasourceUid}/resources/endpoint/${endpointId}/links/${linkName}/action/${actionId}`,
        method: 'POST',
        data: { message },
    });
    return response.data;
}
```

Or using the datasource instance directly:

```typescript
// Assuming you have a datasource instance
const links = await datasource.getResource(`endpoint/${endpointId}/links`);

await datasource.postResource(`endpoint/${endpointId}/links/${linkName}/enable`, {});

await datasource.postResource(`endpoint/${endpointId}/links/${linkName}/disable`, {});

await datasource.postResource(`endpoint/${endpointId}/links/${linkName}/action/${actionId}`, {
    message: { /* optional parameters */ }
});
```

## Expected Test Results

When running the test script, you should see output similar to:

```
============================================================
Starting Link Management Tests
============================================================
ℹ Grafana URL: http://localhost:3000
ℹ Datasource UID: abc123
ℹ Endpoint ID: myEndpoint

============================================================
Test: List Links
============================================================
✓ List Links: Found 3 link(s)
ℹ Links found:
  - udp-in: OK (ENABLED)
  - tcp-tm: OK (ENABLED)
  - tc-out: OK (ENABLED)

============================================================
Test: Get Link 'udp-in'
============================================================
✓ Get Link: Retrieved link 'udp-in'

============================================================
Test: Toggle Link 'udp-in'
============================================================
ℹ Original state: ENABLED
✓ Disable Link: Link 'udp-in' is now disabled
✓ Enable Link: Link 'udp-in' is now enabled
✓ Toggle Link: Link 'udp-in' toggled and restored

============================================================
Test: Reset Link Counters 'udp-in'
============================================================
✓ Reset Link Counters: Counters reset - In: 0, Out: 0

============================================================
Test Summary
============================================================
  [PASS] List Links
  [PASS] Get Link
  [PASS] Toggle Link
  [PASS] Disable Link
  [PASS] Enable Link
  [PASS] Reset Link Counters

Total: 6/6 tests passed
All tests passed!
```

## Troubleshooting

### Common Issues

**"No links found" error:**
- Verify your Yamcs instance has configured links
- Check the endpoint is correctly configured in the datasource
- Ensure the Yamcs server is accessible

**"HTTP 404" or "HTTP 500" errors:**
- Rebuild the plugin: `mage build:backend && pnpm build`
- Restart Grafana: `docker compose restart`
- Check Grafana logs: `docker logs jaops-yamcs-app --tail=50`

**Authentication errors:**
- If Grafana requires authentication, use `--username` and `--password` options
- Check your Grafana authentication settings

**Endpoint not found:**
- Verify the endpoint ID matches your datasource configuration
- Use `--list-endpoints` to see available endpoints

### Debug Mode

Enable debug logging in Grafana to see detailed request/response information:

1. Edit your Grafana configuration or environment variables
2. Set: `GF_LOG_FILTERS: plugin.jaops-yamcs-app:debug`
3. Restart Grafana
4. Check logs: `docker logs jaops-yamcs-app -f`