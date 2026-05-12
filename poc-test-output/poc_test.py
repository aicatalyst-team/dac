#!/usr/bin/env python3
"""AutoPoC — Proof of Concept Test Script for DAC (Dashboard-as-Code)

Tests the deployed DAC service according to the PoC plan:
1. Frontend loads (GET /)
2. List dashboards API (GET /api/v1/dashboards)
3. Get dashboard detail (GET /api/v1/dashboards/Sales%20Analytics)
4. Validate dashboards CLI job (check Kubernetes job status)
5. Version check CLI job (check Kubernetes job status)
"""

import json
import os
import sys
import time
import urllib.request
import urllib.error
import subprocess

# Configuration
SERVICE_URL = os.environ.get("SERVICE_URL", sys.argv[1] if len(sys.argv) > 1 else "")
MAX_RETRIES = 5
RETRY_DELAY = 10  # seconds
NAMESPACE = "dac"

results = []


def test_scenario(name, description, method, path, body=None, expected_status=200,
                  expected_content=None, validate_fn=None, timeout=30):
    """Run a single HTTP test scenario."""
    url = f"{SERVICE_URL.rstrip('/')}{path}"
    start = time.time()

    for attempt in range(MAX_RETRIES):
        try:
            if body:
                data = json.dumps(body).encode("utf-8") if isinstance(body, dict) else body.encode("utf-8")
                req = urllib.request.Request(url, data=data, method=method)
                req.add_header("Content-Type", "application/json")
            else:
                req = urllib.request.Request(url, method=method)

            with urllib.request.urlopen(req, timeout=timeout) as resp:
                status = resp.status
                response_body = resp.read().decode("utf-8")

                if status != expected_status:
                    if attempt < MAX_RETRIES - 1:
                        time.sleep(RETRY_DELAY)
                        continue
                    result = {
                        "scenario_name": name,
                        "status": "fail",
                        "output": response_body[:2000],
                        "error_message": f"Expected status {expected_status}, got {status}",
                        "duration_seconds": round(time.time() - start, 2),
                    }
                    results.append(result)
                    return result

                # Check expected content string
                if expected_content and expected_content not in response_body:
                    result = {
                        "scenario_name": name,
                        "status": "fail",
                        "output": response_body[:2000],
                        "error_message": f"Expected content '{expected_content}' not found in response",
                        "duration_seconds": round(time.time() - start, 2),
                    }
                    results.append(result)
                    return result

                # Run custom validation function if provided
                if validate_fn:
                    validation_error = validate_fn(response_body)
                    if validation_error:
                        result = {
                            "scenario_name": name,
                            "status": "fail",
                            "output": response_body[:2000],
                            "error_message": validation_error,
                            "duration_seconds": round(time.time() - start, 2),
                        }
                        results.append(result)
                        return result

                result = {
                    "scenario_name": name,
                    "status": "pass",
                    "output": response_body[:2000],
                    "error_message": None,
                    "duration_seconds": round(time.time() - start, 2),
                }
                results.append(result)
                return result

        except urllib.error.HTTPError as e:
            response_body = ""
            try:
                response_body = e.read().decode("utf-8")
            except Exception:
                pass
            if attempt < MAX_RETRIES - 1:
                print(f"  Attempt {attempt + 1}/{MAX_RETRIES} failed: HTTP {e.code}. Retrying in {RETRY_DELAY}s...",
                      file=sys.stderr)
                time.sleep(RETRY_DELAY)
            else:
                result = {
                    "scenario_name": name,
                    "status": "fail",
                    "output": response_body[:2000],
                    "error_message": f"HTTP error after {MAX_RETRIES} attempts: {e.code} {e.reason}",
                    "duration_seconds": round(time.time() - start, 2),
                }
                results.append(result)
                return result

        except urllib.error.URLError as e:
            if attempt < MAX_RETRIES - 1:
                print(f"  Attempt {attempt + 1}/{MAX_RETRIES} failed: {e}. Retrying in {RETRY_DELAY}s...",
                      file=sys.stderr)
                time.sleep(RETRY_DELAY)
            else:
                result = {
                    "scenario_name": name,
                    "status": "error",
                    "output": "",
                    "error_message": f"Service unreachable after {MAX_RETRIES} attempts: {e}",
                    "duration_seconds": round(time.time() - start, 2),
                }
                results.append(result)
                return result

        except Exception as e:
            result = {
                "scenario_name": name,
                "status": "error",
                "output": "",
                "error_message": str(e),
                "duration_seconds": round(time.time() - start, 2),
            }
            results.append(result)
            return result


def test_k8s_job(name, description, job_name, expected_log_content=None):
    """Test a Kubernetes job by checking its status and logs."""
    start = time.time()

    try:
        # Get job status
        job_result = subprocess.run(
            ["kubectl", "get", "job", job_name, "-n", NAMESPACE, "-o", "json"],
            capture_output=True, text=True, timeout=15
        )
        if job_result.returncode != 0:
            result = {
                "scenario_name": name,
                "status": "error",
                "output": job_result.stderr[:2000],
                "error_message": f"Failed to get job {job_name}: {job_result.stderr}",
                "duration_seconds": round(time.time() - start, 2),
            }
            results.append(result)
            return result

        job_data = json.loads(job_result.stdout)
        job_status = job_data.get("status", {})
        succeeded = job_status.get("succeeded", 0)

        # Check if job completed successfully
        conditions = job_status.get("conditions", [])
        is_complete = any(
            c.get("type") == "Complete" and c.get("status") == "True"
            for c in conditions
        )

        if not is_complete or succeeded < 1:
            result = {
                "scenario_name": name,
                "status": "fail",
                "output": json.dumps(job_status, indent=2)[:2000],
                "error_message": f"Job {job_name} did not complete successfully. Succeeded: {succeeded}",
                "duration_seconds": round(time.time() - start, 2),
            }
            results.append(result)
            return result

        # Get pod logs
        pods_result = subprocess.run(
            ["kubectl", "get", "pods", "-n", NAMESPACE, "-l",
             f"job-name={job_name}", "-o", "json"],
            capture_output=True, text=True, timeout=15
        )
        log_output = ""
        if pods_result.returncode == 0:
            pods_data = json.loads(pods_result.stdout)
            items = pods_data.get("items", [])
            if items:
                pod_name = items[0]["metadata"]["name"]
                logs_result = subprocess.run(
                    ["kubectl", "logs", pod_name, "-n", NAMESPACE],
                    capture_output=True, text=True, timeout=15
                )
                log_output = logs_result.stdout

        # Check expected log content
        if expected_log_content and expected_log_content not in log_output:
            result = {
                "scenario_name": name,
                "status": "fail",
                "output": log_output[:2000],
                "error_message": f"Expected '{expected_log_content}' not found in job logs",
                "duration_seconds": round(time.time() - start, 2),
            }
            results.append(result)
            return result

        result = {
            "scenario_name": name,
            "status": "pass",
            "output": log_output[:2000] if log_output else f"Job {job_name} completed successfully (succeeded={succeeded})",
            "error_message": None,
            "duration_seconds": round(time.time() - start, 2),
        }
        results.append(result)
        return result

    except Exception as e:
        result = {
            "scenario_name": name,
            "status": "error",
            "output": "",
            "error_message": str(e),
            "duration_seconds": round(time.time() - start, 2),
        }
        results.append(result)
        return result


# ========== TEST SCENARIOS ==========

# Scenario 1: Frontend Loads
print("Running: frontend-loads", file=sys.stderr)
test_scenario(
    name="frontend-loads",
    description="Verify the embedded React frontend is served at the root URL",
    method="GET",
    path="/",
    expected_status=200,
    expected_content="<html",
    timeout=30,
)

# Scenario 2: List Dashboards API
# Source code shows the actual API route is /api/v1/dashboards
print("Running: list-dashboards", file=sys.stderr)


def validate_dashboards_list(body):
    """Validate that the response is a JSON structure with at least one dashboard."""
    try:
        data = json.loads(body)
    except json.JSONDecodeError:
        return "Response is not valid JSON"

    # The API returns {"dashboards": [...]}
    if isinstance(data, dict):
        dashboards = data.get("dashboards", [])
        if isinstance(dashboards, list):
            if len(dashboards) < 1:
                return "Expected at least one dashboard in the list, got empty array"
            # Verify at least one has a name
            has_named = any(d.get("name") for d in dashboards if isinstance(d, dict))
            if not has_named:
                return "No dashboard entries have a 'name' field"
            return None

    if isinstance(data, list):
        if len(data) < 1:
            return "Expected at least one dashboard in the list, got empty array"
        return None

    return "Unexpected response structure"


test_scenario(
    name="list-dashboards",
    description="Verify the API lists available dashboards from the example project",
    method="GET",
    path="/api/v1/dashboards",
    expected_status=200,
    validate_fn=validate_dashboards_list,
    timeout=15,
)

# Scenario 3: Get Dashboard Detail
# The dashboard name is "Sales Analytics" (with space), URL-encoded as "Sales%20Analytics"
print("Running: get-dashboard", file=sys.stderr)


def validate_dashboard_detail(body):
    """Validate that the response contains dashboard details with name, rows, and widgets."""
    try:
        data = json.loads(body)
    except json.JSONDecodeError:
        return "Response is not valid JSON"

    if not isinstance(data, dict):
        return f"Expected JSON object, got {type(data).__name__}"

    # Check for name
    name = data.get("name", "")
    if not name:
        return "Dashboard detail missing 'name' field"

    # Check for rows (dashboard layout structure)
    rows = data.get("rows", [])
    if not isinstance(rows, list) or len(rows) < 1:
        return "Dashboard detail missing 'rows' or rows are empty"

    # Verify at least one row has widgets
    has_widgets = False
    for row in rows:
        if isinstance(row, dict):
            widgets = row.get("widgets", [])
            if isinstance(widgets, list) and len(widgets) > 0:
                has_widgets = True
                break

    if not has_widgets:
        return "Dashboard rows do not contain any widgets"

    return None


test_scenario(
    name="get-dashboard",
    description="Verify a specific dashboard (Sales Analytics) can be loaded via the API",
    method="GET",
    path="/api/v1/dashboards/Sales%20Analytics",
    expected_status=200,
    validate_fn=validate_dashboard_detail,
    timeout=15,
)

# Scenario 4: Validate Dashboards (CLI Job)
print("Running: validate-dashboards", file=sys.stderr)
test_k8s_job(
    name="validate-dashboards",
    description="Verify the validate command works against the example dashboards",
    job_name="dac-validate-dashboards",
    expected_log_content="validated successfully",
)

# Scenario 5: Version Check (CLI Job)
print("Running: version-check", file=sys.stderr)
test_k8s_job(
    name="version-check",
    description="Verify the built binary reports its version correctly",
    job_name="dac-version-check",
)

# ========== END SCENARIOS ==========

# Output results as JSON
print(json.dumps({"results": results}, indent=2))

# Exit with appropriate code
failed = any(r["status"] in ("fail", "error") for r in results)
sys.exit(1 if failed else 0)
