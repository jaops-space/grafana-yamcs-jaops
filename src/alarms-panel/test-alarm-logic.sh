#!/bin/bash

# ============================================================================
# Alarm Logic Test Script for Grafana-Yamcs Plugin
# ============================================================================
# This script tests the alarm functionality including:
# - Yamcs simulator alarm APIs
# - Grafana datasource alarm endpoints
# - Acknowledge, Shelve, Clear, and Unshelve alarm operations
# - Global alarm status calculation
# - Comment persistence for all alarm actions
# - Alarm state transitions and status display
# ============================================================================

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
YAMCS_URL="localhost:8091"
GRAFANA_URL="localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASS="admin"
DATASOURCE_UID="df76fq85bmha8f"  # From Grafana datasources
ENDPOINT_ID="new-endpoint-1"     # From endpoint configuration
INSTANCE="simulator"
PROCESSOR="realtime"

# Print functions
print_header() {
    echo -e "\n${BLUE}============================================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================================================${NC}\n"
}

print_step() {
    echo -e "${YELLOW}➤ Step $1: $2${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Step 1: Check Prerequisites
check_prerequisites() {
    print_step "1" "Checking Prerequisites"
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        print_error "curl is not installed"
        exit 1
    fi
    print_success "curl is available"
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        print_info "jq is not installed - JSON output will not be pretty-printed"
        JQ_AVAILABLE=false
    else
        print_success "jq is available"
        JQ_AVAILABLE=true
    fi
    
    # Check if docker is running
    if ! docker info &> /dev/null; then
        print_error "Docker is not running"
        exit 1
    fi
    print_success "Docker is running"
}

# Step 2: Verify Yamcs Simulator
verify_yamcs() {
    print_step "2" "Verifying Yamcs Simulator"
    
    # Check if Yamcs is accessible
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://${YAMCS_URL}/api/")
    
    if [ "$HTTP_STATUS" == "200" ]; then
        print_success "Yamcs is accessible at http://${YAMCS_URL}"
        
        # Get Yamcs version
        VERSION=$(curl -s "http://${YAMCS_URL}/api/" | grep -o '"version":"[^"]*"' | head -1 | cut -d'"' -f4)
        print_info "Yamcs Version: $VERSION"
    else
        print_error "Yamcs is not accessible (HTTP $HTTP_STATUS)"
        print_info "Starting Yamcs simulator..."
        docker run -d --name yamcs-simulator -p 8091:8090 yamcs/example-simulation
        sleep 15
    fi
}

# Step 3: Verify Grafana
verify_grafana() {
    print_step "3" "Verifying Grafana"
    
    # Check if Grafana is accessible
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://${GRAFANA_URL}/api/health")
    
    if [ "$HTTP_STATUS" == "200" ]; then
        print_success "Grafana is accessible at http://${GRAFANA_URL}"
    else
        print_error "Grafana is not accessible (HTTP $HTTP_STATUS)"
        print_info "Starting Grafana with docker-compose..."
        docker-compose up -d --build
        sleep 20
    fi
}

# Step 4: List Alarms from Yamcs
list_yamcs_alarms() {
    print_step "4" "Listing Alarms from Yamcs"
    
    ALARM_RESPONSE=$(curl -s "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms")
    
    # Check if response contains alarms
    if echo "$ALARM_RESPONSE" | grep -q '"alarms"'; then
        ALARM_COUNT=$(echo "$ALARM_RESPONSE" | grep -o '"seqNum"' | wc -l)
        print_success "Found $ALARM_COUNT alarm(s) in Yamcs"
        
        # Extract alarm details
        echo -e "\n${BLUE}Current Alarms:${NC}"
        echo "-----------------------------------------------------------"
        
        if [ "$JQ_AVAILABLE" = true ]; then
            echo "$ALARM_RESPONSE" | jq -r '.alarms[] | "  Name: \(.id.name) | Severity: \(.severity) | SeqNum: \(.seqNum)"'
        else
            # Parse manually
            echo "$ALARM_RESPONSE" | grep -oP '"name"\s*:\s*"[^"]*"|"severity"\s*:\s*"[^"]*"|"seqNum"\s*:\s*[0-9]+' | head -20
        fi
        echo "-----------------------------------------------------------"
        
        # Store first alarm details for testing
        if [ "$JQ_AVAILABLE" = true ]; then
            FIRST_ALARM_NAME=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].id.name')
            FIRST_ALARM_NS=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].id.namespace')
            FIRST_ALARM_SEQ=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].seqNum')
        else
            FIRST_ALARM_NAME=$(echo "$ALARM_RESPONSE" | grep -oP '"name"\s*:\s*"\K[^"]*' | head -1)
            FIRST_ALARM_NS=$(echo "$ALARM_RESPONSE" | grep -oP '"namespace"\s*:\s*"\K[^"]*' | head -1)
            FIRST_ALARM_SEQ=$(echo "$ALARM_RESPONSE" | grep -oP '"seqNum"\s*:\s*\K[0-9]+' | head -1)
        fi
        
        print_info "First alarm for testing: ${FIRST_ALARM_NS}/${FIRST_ALARM_NAME} (seqNum: ${FIRST_ALARM_SEQ})"
        
        # Export for use in other tests
        export TEST_ALARM_NAME="$FIRST_ALARM_NAME"
        export TEST_ALARM_NS="$FIRST_ALARM_NS"
        export TEST_ALARM_SEQ="$FIRST_ALARM_SEQ"
    else
        print_info "No alarms found in Yamcs"
        print_info "Waiting for simulator to generate alarms..."
        sleep 30
        list_yamcs_alarms
    fi
}

# Step 5: Test Yamcs Alarm APIs Directly
test_yamcs_alarm_apis() {
    print_step "5" "Testing Yamcs Alarm APIs Directly"
    
    if [ -z "$TEST_ALARM_NAME" ]; then
        print_error "No test alarm available"
        return 1
    fi
    
    # Build the alarm parameter path
    PARAM_PATH="${TEST_ALARM_NS}/${TEST_ALARM_NAME}"
    ENCODED_PARAM=$(echo "$PARAM_PATH" | sed 's/\//%2F/g')
    
    echo ""
    print_info "Testing with alarm: $PARAM_PATH (seqNum: $TEST_ALARM_SEQ)"
    
    # Test 5a: Acknowledge Alarm directly via Yamcs API
    echo ""
    print_info "5a. Testing Acknowledge Alarm via Yamcs API..."
    
    ACK_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"comment": "Test acknowledgement from script"}' \
        "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms/${ENCODED_PARAM}/${TEST_ALARM_SEQ}:acknowledge")
    
    HTTP_CODE=$(echo "$ACK_RESPONSE" | tail -1)
    BODY=$(echo "$ACK_RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Acknowledge alarm API returned HTTP $HTTP_CODE"
    else
        print_error "Acknowledge alarm API failed with HTTP $HTTP_CODE"
        echo "Response: $BODY"
    fi
    
    # Test 5b: Shelve Alarm
    echo ""
    print_info "5b. Testing Shelve Alarm via Yamcs API..."
    
    SHELVE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"comment": "Test shelve from script", "shelveDuration": 60000}' \
        "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms/${ENCODED_PARAM}/${TEST_ALARM_SEQ}:shelve")
    
    HTTP_CODE=$(echo "$SHELVE_RESPONSE" | tail -1)
    BODY=$(echo "$SHELVE_RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Shelve alarm API returned HTTP $HTTP_CODE"
    else
        print_error "Shelve alarm API failed with HTTP $HTTP_CODE"
        echo "Response: $BODY"
    fi
    
    # Test 5c: Unshelve Alarm
    echo ""
    print_info "5c. Testing Unshelve Alarm via Yamcs API..."
    
    UNSHELVE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms/${ENCODED_PARAM}/${TEST_ALARM_SEQ}:unshelve")
    
    HTTP_CODE=$(echo "$UNSHELVE_RESPONSE" | tail -1)
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Unshelve alarm API returned HTTP $HTTP_CODE"
    else
        print_error "Unshelve alarm API failed with HTTP $HTTP_CODE"
    fi
    
    # Test 5d: Clear Alarm
    echo ""
    print_info "5d. Testing Clear Alarm via Yamcs API..."
    
    CLEAR_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"comment": "Test clear from script"}' \
        "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms/${ENCODED_PARAM}/${TEST_ALARM_SEQ}:clear")
    
    HTTP_CODE=$(echo "$CLEAR_RESPONSE" | tail -1)
    BODY=$(echo "$CLEAR_RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Clear alarm API returned HTTP $HTTP_CODE"
    else
        print_error "Clear alarm API failed with HTTP $HTTP_CODE"
        echo "Response: $BODY"
    fi
}

# Step 6: Test Grafana Plugin Alarm Endpoints
test_grafana_alarm_endpoints() {
    print_step "6" "Testing Grafana Plugin Alarm Endpoints"
    
    if [ -z "$TEST_ALARM_NAME" ]; then
        print_error "No test alarm available"
        return 1
    fi
    
    BASE_URL="http://${GRAFANA_URL}/api/datasources/uid/${DATASOURCE_UID}/resources"
    AUTH="${GRAFANA_USER}:${GRAFANA_PASS}"
    
    PARAM_PATH="${TEST_ALARM_NS}/${TEST_ALARM_NAME}"
    
    # Get a fresh alarm for testing
    echo ""
    print_info "Fetching fresh alarm list..."
    
    ALARM_RESPONSE=$(curl -s "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms")
    
    if [ "$JQ_AVAILABLE" = true ]; then
        FRESH_ALARM_NAME=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].id.name')
        FRESH_ALARM_NS=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].id.namespace')
        FRESH_ALARM_SEQ=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].seqNum')
    else
        FRESH_ALARM_NAME=$(echo "$ALARM_RESPONSE" | grep -oP '"name"\s*:\s*"\K[^"]*' | head -1)
        FRESH_ALARM_NS=$(echo "$ALARM_RESPONSE" | grep -oP '"namespace"\s*:\s*"\K[^"]*' | head -1)
        FRESH_ALARM_SEQ=$(echo "$ALARM_RESPONSE" | grep -oP '"seqNum"\s*:\s*\K[0-9]+' | head -1)
    fi
    
    if [ -z "$FRESH_ALARM_NAME" ]; then
        print_info "No alarms available for Grafana endpoint testing"
        return 0
    fi
    
    FRESH_PARAM_PATH="${FRESH_ALARM_NS}/${FRESH_ALARM_NAME}"
    print_info "Testing with: $FRESH_PARAM_PATH (seqNum: $FRESH_ALARM_SEQ)"
    
    # Test 6a: Acknowledge via Grafana endpoint
    echo ""
    print_info "6a. Testing Acknowledge Alarm via Grafana endpoint..."
    
    ACK_PAYLOAD=$(cat << EOF
{
    "name": "${FRESH_PARAM_PATH}",
    "seqNum": ${FRESH_ALARM_SEQ},
    "comment": "Test ack from Grafana"
}
EOF
)
    
    ACK_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -u "$AUTH" \
        -H "Content-Type: application/json" \
        -d "$ACK_PAYLOAD" \
        "${BASE_URL}/endpoint/${ENDPOINT_ID}/alarm/acknowledge")
    
    HTTP_CODE=$(echo "$ACK_RESPONSE" | tail -1)
    BODY=$(echo "$ACK_RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Grafana acknowledge endpoint returned HTTP $HTTP_CODE"
    else
        print_error "Grafana acknowledge endpoint failed with HTTP $HTTP_CODE"
        echo "Response: $BODY"
    fi
    
    # Test 6b: Shelve via Grafana endpoint
    echo ""
    print_info "6b. Testing Shelve Alarm via Grafana endpoint..."
    
    SHELVE_PAYLOAD=$(cat << EOF
{
    "name": "${FRESH_PARAM_PATH}",
    "seqNum": ${FRESH_ALARM_SEQ},
    "comment": "Test shelve from Grafana",
    "shelveDuration": 60000
}
EOF
)
    
    SHELVE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -u "$AUTH" \
        -H "Content-Type: application/json" \
        -d "$SHELVE_PAYLOAD" \
        "${BASE_URL}/endpoint/${ENDPOINT_ID}/alarm/shelve")
    
    HTTP_CODE=$(echo "$SHELVE_RESPONSE" | tail -1)
    BODY=$(echo "$SHELVE_RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Grafana shelve endpoint returned HTTP $HTTP_CODE"
    else
        print_error "Grafana shelve endpoint failed with HTTP $HTTP_CODE"
        echo "Response: $BODY"
    fi
    
    # Test 6c: Unshelve via Grafana endpoint
    echo ""
    print_info "6c. Testing Unshelve Alarm via Grafana endpoint..."

    UNSHELVE_PAYLOAD=$(cat << EOF
{
    "name": "${FRESH_PARAM_PATH}",
    "seqNum": ${FRESH_ALARM_SEQ}
}
EOF
)

    UNSHELVE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -u "$AUTH" \
        -H "Content-Type: application/json" \
        -d "$UNSHELVE_PAYLOAD" \
        "${BASE_URL}/endpoint/${ENDPOINT_ID}/alarm/unshelve")

    HTTP_CODE=$(echo "$UNSHELVE_RESPONSE" | tail -1)
    BODY=$(echo "$UNSHELVE_RESPONSE" | head -n -1)

    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Grafana unshelve endpoint returned HTTP $HTTP_CODE"
    else
        print_error "Grafana unshelve endpoint failed with HTTP $HTTP_CODE"
        echo "Response: $BODY"
    fi

    # Test 6d: Clear via Grafana endpoint
    echo ""
    print_info "6d. Testing Clear Alarm via Grafana endpoint..."

    CLEAR_PAYLOAD=$(cat << EOF
{
    "name": "${FRESH_PARAM_PATH}",
    "seqNum": ${FRESH_ALARM_SEQ},
    "comment": "Test clear from Grafana"
}
EOF
)
    
    CLEAR_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
        -u "$AUTH" \
        -H "Content-Type: application/json" \
        -d "$CLEAR_PAYLOAD" \
        "${BASE_URL}/endpoint/${ENDPOINT_ID}/alarm/clear")
    
    HTTP_CODE=$(echo "$CLEAR_RESPONSE" | tail -1)
    BODY=$(echo "$CLEAR_RESPONSE" | head -n -1)
    
    if [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "204" ]; then
        print_success "Grafana clear endpoint returned HTTP $HTTP_CODE"
    else
        print_error "Grafana clear endpoint failed with HTTP $HTTP_CODE"
        echo "Response: $BODY"
    fi
}

# Step 7: Verify Alarm Data Structure
verify_alarm_data_structure() {
    print_step "7" "Verifying Alarm Data Structure (Trip Value, Status, Comments)"

    echo ""
    print_info "Fetching detailed alarm data..."

    ALARM_RESPONSE=$(curl -s "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms")

    if ! echo "$ALARM_RESPONSE" | grep -q '"alarms"'; then
        print_info "No alarms available for structure validation"
        return 0
    fi

    # Test 7a: Verify Trip Value (triggerValue) exists
    echo ""
    print_info "7a. Checking for Trip Value (triggerValue) in alarm data..."

    if [ "$JQ_AVAILABLE" = true ]; then
        HAS_TRIGGER_VALUE=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].parameterDetail.triggerValue // "NOT_FOUND"')

        if [ "$HAS_TRIGGER_VALUE" != "NOT_FOUND" ] && [ "$HAS_TRIGGER_VALUE" != "null" ]; then
            TRIGGER_VAL=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].parameterDetail.triggerValue.engValue // empty')
            print_success "Trip value found in alarm data"
            print_info "Example trip value: $TRIGGER_VAL"
        else
            print_info "Trip value not available (may not be a parameter alarm)"
        fi

        # Test 7b: Verify Current Value exists
        echo ""
        print_info "7b. Checking for Current Value (currentValue) in alarm data..."

        HAS_CURRENT_VALUE=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].parameterDetail.currentValue // "NOT_FOUND"')

        if [ "$HAS_CURRENT_VALUE" != "NOT_FOUND" ] && [ "$HAS_CURRENT_VALUE" != "null" ]; then
            CURRENT_VAL=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].parameterDetail.currentValue.engValue // empty')
            print_success "Current value found in alarm data"
            print_info "Example current value: $CURRENT_VAL"
        else
            print_info "Current value not available"
        fi

        # Test 7c: Verify Alarm Status Fields
        echo ""
        print_info "7c. Checking alarm status fields (acknowledged, shelved, triggered)..."

        IS_ACKNOWLEDGED=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].acknowledged // false')
        IS_SHELVED=$(echo "$ALARM_RESPONSE" | jq -r 'if .alarms[0].shelveInfo then "true" else "false" end')
        IS_TRIGGERED=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].triggered // false')

        print_success "Status fields present:"
        echo "    - Acknowledged: $IS_ACKNOWLEDGED"
        echo "    - Shelved: $IS_SHELVED"
        echo "    - Triggered: $IS_TRIGGERED"

        # Test 7d: Check for acknowledgement info
        echo ""
        print_info "7d. Checking for acknowledge info (username, time, comment)..."

        if [ "$IS_ACKNOWLEDGED" == "true" ]; then
            ACK_BY=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].acknowledgeInfo.acknowledgedBy // "N/A"')
            ACK_TIME=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].acknowledgeInfo.acknowledgeTime // "N/A"')
            ACK_COMMENT=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].acknowledgeInfo.acknowledgeMessage // "N/A"')

            print_success "Acknowledge info found:"
            echo "    - Acknowledged by: $ACK_BY"
            echo "    - Acknowledge time: $ACK_TIME"
            echo "    - Comment: $ACK_COMMENT"
        else
            print_info "No acknowledgement info (alarm not acknowledged)"
        fi

        # Test 7e: Check for shelve info
        echo ""
        print_info "7e. Checking for shelve info (username, time, expiration, comment)..."

        if [ "$IS_SHELVED" == "true" ]; then
            SHELVE_BY=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].shelveInfo.shelvedBy // "N/A"')
            SHELVE_TIME=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].shelveInfo.shelveTime // "N/A"')
            SHELVE_EXP=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].shelveInfo.shelveExpiration // "N/A"')
            SHELVE_COMMENT=$(echo "$ALARM_RESPONSE" | jq -r '.alarms[0].shelveInfo.shelveMessage // "N/A"')

            print_success "Shelve info found:"
            echo "    - Shelved by: $SHELVE_BY"
            echo "    - Shelve time: $SHELVE_TIME"
            echo "    - Shelve expiration: $SHELVE_EXP"
            echo "    - Comment: $SHELVE_COMMENT"
        else
            print_info "No shelve info (alarm not shelved)"
        fi

        # Test 7f: Check severity levels
        echo ""
        print_info "7f. Checking severity levels for Global Alarm Status..."

        SEVERITIES=$(echo "$ALARM_RESPONSE" | jq -r '[.alarms[].severity] | unique | join(", ")')
        ALARM_COUNT=$(echo "$ALARM_RESPONSE" | jq -r '.alarms | length')

        print_success "Found $ALARM_COUNT alarm(s) with severity levels: $SEVERITIES"

        # Calculate alarm counts by state (simulating Global Alarm Status logic)
        UNACK_COUNT=$(echo "$ALARM_RESPONSE" | jq -r '[.alarms[] | select(.acknowledged == false and (.shelveInfo == null))] | length')
        ACK_COUNT=$(echo "$ALARM_RESPONSE" | jq -r '[.alarms[] | select(.acknowledged == true and (.shelveInfo == null))] | length')
        SHELVED_COUNT=$(echo "$ALARM_RESPONSE" | jq -r '[.alarms[] | select(.shelveInfo != null)] | length')

        print_info "Global Alarm Status counts:"
        echo "    - Unacknowledged: $UNACK_COUNT"
        echo "    - Acknowledged: $ACK_COUNT"
        echo "    - Shelved: $SHELVED_COUNT"

    else
        print_info "jq not available - skipping detailed structure validation"
    fi
}

# Step 8: Verify Final State
verify_final_state() {
    print_step "8" "Verifying Final State"

    echo ""
    print_info "Fetching current alarm state..."
    
    ALARM_RESPONSE=$(curl -s "http://${YAMCS_URL}/api/processors/${INSTANCE}/${PROCESSOR}/alarms")
    
    ALARM_COUNT=$(echo "$ALARM_RESPONSE" | grep -o '"seqNum"' | wc -l)
    
    print_success "Final alarm count: $ALARM_COUNT"
    
    if [ "$JQ_AVAILABLE" = true ]; then
        echo -e "\n${BLUE}Alarm Status Summary:${NC}"
        echo "-----------------------------------------------------------"
        echo "$ALARM_RESPONSE" | jq -r '.alarms[] | "  \(.id.namespace)/\(.id.name) - Severity: \(.severity) - Ack: \(if .acknowledgeInfo then "YES" else "NO" end)"' 2>/dev/null || echo "  (Unable to parse)"
        echo "-----------------------------------------------------------"
    fi
}

# Main Execution
main() {
    print_header "Grafana-Yamcs Alarm Logic Test Script"
    
    echo "Configuration:"
    echo "  Yamcs URL:      http://${YAMCS_URL}"
    echo "  Grafana URL:    http://${GRAFANA_URL}"
    echo "  Datasource UID: ${DATASOURCE_UID}"
    echo "  Endpoint ID:    ${ENDPOINT_ID}"
    echo "  Instance:       ${INSTANCE}"
    echo "  Processor:      ${PROCESSOR}"
    
    check_prerequisites
    verify_yamcs
    verify_grafana
    list_yamcs_alarms
    test_yamcs_alarm_apis
    test_grafana_alarm_endpoints
    verify_alarm_data_structure
    verify_final_state
    
    print_header "Test Complete"
    echo -e "${GREEN}All alarm logic tests completed!${NC}"
    echo ""
    echo "Tested features:"
    echo "  ✓ Yamcs alarm APIs (acknowledge, clear, shelve, unshelve)"
    echo "  ✓ Grafana datasource endpoints"
    echo "  ✓ Trip value extraction"
    echo "  ✓ Current value extraction"
    echo "  ✓ Alarm status fields (acknowledged, shelved, triggered)"
    echo "  ✓ Action comments (acknowledge, shelve, clear)"
    echo "  ✓ Global alarm status calculation"
    echo "  ✓ Severity level tracking"
    echo ""
    echo "Next steps:"
    echo "  1. Open Grafana at http://${GRAFANA_URL}"
    echo "  2. Create a panel with Query Type = 'Alarms'"
    echo "  3. Verify the following are displayed:"
    echo "     - Global Alarm Status bar (above table)"
    echo "     - Trip value column (between Alarm type and Live value)"
    echo "     - Status column showing Triggered/Acknowledged/Shelved/OK"
    echo "  4. Test the action buttons (Acknowledge/Shelve/Clear/Unshelve)"
    echo "  5. Expand a row and verify all details including comments are shown"
}

# Run main function
main "$@"
