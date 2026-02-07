#!/usr/bin/env python3
"""
Test script for Yamcs Link Management API endpoints.

This script tests the link management feature that allows enabling/disabling
Yamcs links from Grafana displays.

Prerequisites:
- Grafana running on localhost:3000 (default)
- A Yamcs datasource configured with at least one endpoint
- The Yamcs server accessible with configured links

Usage:
    python3 test_link_management.py [--grafana-url URL] [--datasource-uid UID] [--endpoint ENDPOINT]

Examples:
    # Use defaults (will prompt for datasource UID)
    python3 test_link_management.py
    
    # Specify all options
    python3 test_link_management.py --grafana-url http://localhost:3000 --datasource-uid abc123 --endpoint myEndpoint
"""

import argparse
import json
import sys
import time
from typing import Any, Dict, List, Optional

try:
    import requests
except ImportError:
    print("Error: 'requests' library is required. Install it with: pip install requests")
    sys.exit(1)


class Colors:
    """ANSI color codes for terminal output."""
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    RESET = '\033[0m'
    BOLD = '\033[1m'


def print_header(text: str):
    """Print a formatted header."""
    print(f"\n{Colors.BOLD}{Colors.BLUE}{'='*60}{Colors.RESET}")
    print(f"{Colors.BOLD}{Colors.BLUE}{text}{Colors.RESET}")
    print(f"{Colors.BOLD}{Colors.BLUE}{'='*60}{Colors.RESET}")


def print_success(text: str):
    """Print success message."""
    print(f"{Colors.GREEN}✓ {text}{Colors.RESET}")


def print_error(text: str):
    """Print error message."""
    print(f"{Colors.RED}✗ {text}{Colors.RESET}")


def print_warning(text: str):
    """Print warning message."""
    print(f"{Colors.YELLOW}⚠ {text}{Colors.RESET}")


def print_info(text: str):
    """Print info message."""
    print(f"{Colors.BLUE}ℹ {text}{Colors.RESET}")


class LinkManagementTester:
    """Test class for Yamcs Link Management API."""

    def __init__(self, grafana_url: str, datasource_uid: str, endpoint_id: str,
                 username: Optional[str] = None, password: Optional[str] = None):
        self.grafana_url = grafana_url.rstrip('/')
        self.datasource_uid = datasource_uid
        self.endpoint_id = endpoint_id
        self.session = requests.Session()
        
        # Set up authentication if provided
        if username and password:
            self.session.auth = (username, password)
        
        self.base_url = f"{self.grafana_url}/api/datasources/uid/{datasource_uid}/resources"
        self.test_results: List[Dict[str, Any]] = []

    def _make_request(self, method: str, path: str, data: Optional[Dict] = None) -> requests.Response:
        """Make an HTTP request to the Grafana datasource resource API."""
        url = f"{self.base_url}/{path}"
        headers = {'Content-Type': 'application/json'}
        
        if method.upper() == 'GET':
            return self.session.get(url, headers=headers, timeout=30)
        elif method.upper() == 'POST':
            return self.session.post(url, headers=headers, json=data, timeout=30)
        else:
            raise ValueError(f"Unsupported HTTP method: {method}")

    def _record_test(self, name: str, passed: bool, details: str = ""):
        """Record a test result."""
        self.test_results.append({
            'name': name,
            'passed': passed,
            'details': details
        })
        if passed:
            print_success(f"{name}: {details}" if details else name)
        else:
            print_error(f"{name}: {details}" if details else name)

    def test_list_links(self) -> List[Dict]:
        """Test listing all links for an endpoint."""
        print_header("Test: List Links")
        
        try:
            response = self._make_request('GET', f"endpoint/{self.endpoint_id}/links")
            
            if response.status_code == 200:
                links = response.json()
                self._record_test("List Links", True, f"Found {len(links)} link(s)")
                
                if links:
                    print_info("Links found:")
                    for link in links:
                        status = "DISABLED" if link.get('disabled') else "ENABLED"
                        print(f"  - {link.get('name')}: {link.get('status')} ({status})")
                        if link.get('actions'):
                            print(f"    Actions: {[a.get('id') for a in link.get('actions', [])]}")
                
                return links
            else:
                self._record_test("List Links", False, f"HTTP {response.status_code}: {response.text}")
                return []
                
        except Exception as e:
            self._record_test("List Links", False, str(e))
            return []

    def test_get_link(self, link_name: str) -> Optional[Dict]:
        """Test getting a specific link."""
        print_header(f"Test: Get Link '{link_name}'")
        
        try:
            response = self._make_request('GET', f"endpoint/{self.endpoint_id}/links/{link_name}")
            
            if response.status_code == 200:
                link = response.json()
                self._record_test("Get Link", True, f"Retrieved link '{link_name}'")
                print_info(f"Link details: {json.dumps(link, indent=2)}")
                return link
            else:
                self._record_test("Get Link", False, f"HTTP {response.status_code}: {response.text}")
                return None
                
        except Exception as e:
            self._record_test("Get Link", False, str(e))
            return None

    def test_disable_link(self, link_name: str) -> bool:
        """Test disabling a link."""
        print_header(f"Test: Disable Link '{link_name}'")
        
        try:
            response = self._make_request('POST', f"endpoint/{self.endpoint_id}/links/{link_name}/disable")
            
            if response.status_code == 200:
                link = response.json()
                if link.get('disabled'):
                    self._record_test("Disable Link", True, f"Link '{link_name}' is now disabled")
                    return True
                else:
                    self._record_test("Disable Link", False, "Link status is not disabled after request")
                    return False
            else:
                self._record_test("Disable Link", False, f"HTTP {response.status_code}: {response.text}")
                return False
                
        except Exception as e:
            self._record_test("Disable Link", False, str(e))
            return False

    def test_enable_link(self, link_name: str) -> bool:
        """Test enabling a link."""
        print_header(f"Test: Enable Link '{link_name}'")
        
        try:
            response = self._make_request('POST', f"endpoint/{self.endpoint_id}/links/{link_name}/enable")
            
            if response.status_code == 200:
                link = response.json()
                if not link.get('disabled'):
                    self._record_test("Enable Link", True, f"Link '{link_name}' is now enabled")
                    return True
                else:
                    self._record_test("Enable Link", False, "Link is still disabled after enable request")
                    return False
            else:
                self._record_test("Enable Link", False, f"HTTP {response.status_code}: {response.text}")
                return False
                
        except Exception as e:
            self._record_test("Enable Link", False, str(e))
            return False

    def test_reset_link_counters(self, link_name: str) -> bool:
        """Test resetting link counters."""
        print_header(f"Test: Reset Link Counters '{link_name}'")
        
        try:
            response = self._make_request('POST', f"endpoint/{self.endpoint_id}/links/{link_name}/reset")
            
            if response.status_code == 200:
                link = response.json()
                self._record_test("Reset Link Counters", True, 
                    f"Counters reset - In: {link.get('dataInCount', 0)}, Out: {link.get('dataOutCount', 0)}")
                return True
            else:
                self._record_test("Reset Link Counters", False, f"HTTP {response.status_code}: {response.text}")
                return False
                
        except Exception as e:
            self._record_test("Reset Link Counters", False, str(e))
            return False

    def test_run_link_action(self, link_name: str, action_id: str, message: Optional[Dict] = None) -> bool:
        """Test running a link action."""
        print_header(f"Test: Run Link Action '{action_id}' on '{link_name}'")
        
        try:
            data = {'message': message} if message else {}
            response = self._make_request('POST', 
                f"endpoint/{self.endpoint_id}/links/{link_name}/action/{action_id}", data)
            
            if response.status_code == 200:
                result = response.json()
                self._record_test("Run Link Action", True, f"Action '{action_id}' executed successfully")
                if result:
                    print_info(f"Action result: {json.dumps(result, indent=2)}")
                return True
            else:
                self._record_test("Run Link Action", False, f"HTTP {response.status_code}: {response.text}")
                return False
                
        except Exception as e:
            self._record_test("Run Link Action", False, str(e))
            return False

    def test_toggle_link(self, link_name: str) -> bool:
        """Test toggling a link (disable then enable)."""
        print_header(f"Test: Toggle Link '{link_name}'")
        
        # First, get current state
        link = self.test_get_link(link_name)
        if not link:
            return False
        
        original_state = link.get('disabled', False)
        print_info(f"Original state: {'DISABLED' if original_state else 'ENABLED'}")
        
        # Toggle to opposite state
        if original_state:
            success = self.test_enable_link(link_name)
        else:
            success = self.test_disable_link(link_name)
        
        if not success:
            return False
        
        # Wait a moment
        time.sleep(1)
        
        # Toggle back
        if original_state:
            success = self.test_disable_link(link_name)
        else:
            success = self.test_enable_link(link_name)
        
        if success:
            self._record_test("Toggle Link", True, f"Link '{link_name}' toggled and restored")
        
        return success

    def run_all_tests(self, skip_actions: bool = False) -> bool:
        """Run all link management tests."""
        print_header("Starting Link Management Tests")
        print_info(f"Grafana URL: {self.grafana_url}")
        print_info(f"Datasource UID: {self.datasource_uid}")
        print_info(f"Endpoint ID: {self.endpoint_id}")
        
        # Test 1: List all links
        links = self.test_list_links()
        
        if not links:
            print_warning("No links found. Make sure the Yamcs instance has configured links.")
            return False
        
        # Use the first link for further tests
        test_link = links[0]
        link_name = test_link.get('name')
        
        print_info(f"\nUsing link '{link_name}' for detailed tests")
        
        # Test 2: Get specific link
        self.test_get_link(link_name)
        
        # Test 3: Toggle test (disable/enable cycle)
        self.test_toggle_link(link_name)
        
        # Test 4: Reset counters
        self.test_reset_link_counters(link_name)
        
        # Test 5: Run actions (if available and not skipped)
        if not skip_actions and test_link.get('actions'):
            for action in test_link.get('actions', []):
                if action.get('enabled'):
                    self.test_run_link_action(link_name, action.get('id'))
                    break
            else:
                print_warning("No enabled actions found on the test link")
        elif not skip_actions:
            print_warning("No actions available on the test link")
        
        # Print summary
        self.print_summary()
        
        return all(r['passed'] for r in self.test_results)

    def print_summary(self):
        """Print test summary."""
        print_header("Test Summary")
        
        passed = sum(1 for r in self.test_results if r['passed'])
        failed = len(self.test_results) - passed
        
        for result in self.test_results:
            status = f"{Colors.GREEN}PASS{Colors.RESET}" if result['passed'] else f"{Colors.RED}FAIL{Colors.RESET}"
            print(f"  [{status}] {result['name']}")
        
        print(f"\n{Colors.BOLD}Total: {passed}/{len(self.test_results)} tests passed{Colors.RESET}")
        
        if failed > 0:
            print(f"{Colors.RED}Some tests failed!{Colors.RESET}")
        else:
            print(f"{Colors.GREEN}All tests passed!{Colors.RESET}")


def get_available_endpoints(grafana_url: str, datasource_uid: str, 
                           username: Optional[str] = None, password: Optional[str] = None) -> Dict:
    """Fetch available endpoints from the datasource."""
    session = requests.Session()
    if username and password:
        session.auth = (username, password)
    
    url = f"{grafana_url}/api/datasources/uid/{datasource_uid}/resources/fetch/endpoints"
    try:
        response = session.get(url, timeout=10)
        if response.status_code == 200:
            return response.json()
    except Exception as e:
        print_error(f"Failed to fetch endpoints: {e}")
    return {}


def main():
    parser = argparse.ArgumentParser(
        description='Test Yamcs Link Management API endpoints',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Interactive mode - will prompt for required values
  python3 test_link_management.py

  # Specify all options
  python3 test_link_management.py --grafana-url http://localhost:3000 \\
      --datasource-uid my-yamcs-datasource --endpoint myEndpoint

  # With authentication
  python3 test_link_management.py --username admin --password admin \\
      --datasource-uid my-yamcs-datasource --endpoint myEndpoint
        """
    )
    
    parser.add_argument('--grafana-url', '-g', default='http://localhost:3000',
                       help='Grafana URL (default: http://localhost:3000)')
    parser.add_argument('--datasource-uid', '-d',
                       help='Yamcs datasource UID')
    parser.add_argument('--endpoint', '-e',
                       help='Endpoint ID to test')
    parser.add_argument('--username', '-u',
                       help='Grafana username for authentication')
    parser.add_argument('--password', '-p',
                       help='Grafana password for authentication')
    parser.add_argument('--skip-actions', action='store_true',
                       help='Skip testing link actions')
    parser.add_argument('--list-endpoints', action='store_true',
                       help='List available endpoints and exit')
    
    args = parser.parse_args()
    
    # If datasource UID not provided, prompt for it
    datasource_uid = args.datasource_uid
    if not datasource_uid:
        print_info("Datasource UID not provided.")
        print_info("You can find it in Grafana: Connections > Data sources > Your Yamcs datasource > UID in URL")
        datasource_uid = input("Enter datasource UID: ").strip()
        if not datasource_uid:
            print_error("Datasource UID is required")
            sys.exit(1)
    
    # List endpoints if requested
    if args.list_endpoints:
        print_header("Available Endpoints")
        endpoints = get_available_endpoints(args.grafana_url, datasource_uid, 
                                           args.username, args.password)
        if endpoints:
            for eid, info in endpoints.items():
                status = "ONLINE" if info.get('online') else "OFFLINE"
                print(f"  - {eid}: {info.get('name', 'Unknown')} [{status}]")
        else:
            print_warning("No endpoints found or unable to fetch endpoints")
        sys.exit(0)
    
    # If endpoint not provided, try to fetch and prompt
    endpoint_id = args.endpoint
    if not endpoint_id:
        print_info("Fetching available endpoints...")
        endpoints = get_available_endpoints(args.grafana_url, datasource_uid,
                                           args.username, args.password)
        
        if endpoints:
            print_info("Available endpoints:")
            endpoint_list = list(endpoints.keys())
            for i, (eid, info) in enumerate(endpoints.items()):
                status = "ONLINE" if info.get('online') else "OFFLINE"
                print(f"  {i+1}. {eid}: {info.get('name', 'Unknown')} [{status}]")
            
            choice = input("\nEnter endpoint number or ID: ").strip()
            try:
                idx = int(choice) - 1
                if 0 <= idx < len(endpoint_list):
                    endpoint_id = endpoint_list[idx]
                else:
                    endpoint_id = choice
            except ValueError:
                endpoint_id = choice
        else:
            endpoint_id = input("Enter endpoint ID: ").strip()
        
        if not endpoint_id:
            print_error("Endpoint ID is required")
            sys.exit(1)
    
    # Run tests
    tester = LinkManagementTester(
        grafana_url=args.grafana_url,
        datasource_uid=datasource_uid,
        endpoint_id=endpoint_id,
        username=args.username,
        password=args.password
    )
    
    success = tester.run_all_tests(skip_actions=args.skip_actions)
    sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()
