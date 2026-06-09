---
name: ecs-linux-diagnostic
description: Comprehensive Linux ECS instance diagnostics with full report. Supports both local execution (when running inside the ECS instance) and remote execution (targeting instances by ID).
---

# ECS Linux Baseline Diagnostic

## Overview
This skill provides comprehensive baseline diagnostics for Alibaba Cloud Linux ECS instances using the `ecsgo-helper` plugin. It supports two execution modes:

- **Local Mode**: When the agent runs directly inside an ECS instance, invoke diagnostics locally via `acs-plugin-manager` and read output directly from stdout.
- **Remote Mode**: When targeting a remote ECS instance by ID, use the bundled Node.js script to execute diagnostics via Cloud Assistant and retrieve the report via OSS.

## When to Use This Skill
Use this skill when:
- Performing routine health checks on Linux ECS instances
- Diagnosing performance or configuration issues on Linux instances
- Running automated baseline diagnostics as part of operational procedures
- Investigating system anomalies or unexpected behavior

## Available Scripts
- **`scripts/ecs-diagnostic-cli.js`** — Executes Linux baseline diagnostics on remote ECS instances and retrieves the complete diagnostic report via OSS. Run with `--help` for usage details.

## Local Execution Mode
Use this mode when the agent is running directly inside the ECS instance to be diagnosed.

### Prerequisites (Local)
- Cloud Assistant agent installed (`aliyun-service` running)
- The `acs-plugin-manager` command available in PATH or at standard locations

### Local Diagnostic Command

Execute the `ecsgo-helper` plugin directly using `acs-plugin-manager`:

```bash
# Basic diagnostic (outputs JSON report to stdout)
acs-plugin-manager --exec --plugin ecsgo-helper --separator=' ' --params="baseline-diagnostic"

# With time range for log analysis
acs-plugin-manager --exec --plugin ecsgo-helper --separator=' ' --params="baseline-diagnostic --since '2026-01-13T00:00:00Z' --until '2026-01-14T00:00:00Z'"
```

### Finding acs-plugin-manager
If `acs-plugin-manager` is not in PATH, search for it in standard Cloud Assistant installation directories:

```bash
# Check standard locations
ls /usr/local/share/aliyun-assist/*/acs-plugin-manager
ls /opt/local/share/aliyun-assist/*/acs-plugin-manager
```

### Local Output Handling

The diagnostic output is a JSON report written directly to stdout. Capture and parse it:

```bash
# Capture to file
acs-plugin-manager --exec --plugin ecsgo-helper --separator=' ' --params="baseline-diagnostic" > diagnostic-report.json

# Or pipe directly for analysis
acs-plugin-manager --exec --plugin ecsgo-helper --separator=' ' --params="baseline-diagnostic" | jq '.analysis'
```

### Local Exit Codes

| Code | Meaning |
|------|---------|  
| 0 | Success |
| 121 | `/bin/sh` not found or not executable |
| 123 | `acs-plugin-manager` not found |

## Remote Execution Mode

Use this mode when targeting a remote ECS instance specified by instance ID.

### Prerequisites (Remote)
- Access to Alibaba Cloud ECS with appropriate permissions
- Cloud Assistant agent installed and running on target instances
- Valid region configuration for ECS API calls
- Access to OSS bucket for storing diagnostic reports
- Proper IAM permissions for ECS, OSS, and RAM services
- Node.js 20 or higher (same runtime as the agent CLI)

### Remote Workflow

#### 1. Identify Target Instance
First, identify the Linux ECS instance to diagnose:
```bash
aliyun ecs DescribeInstances --InstanceIds '["INSTANCE_ID"]'
```

#### 2. Execute Complete Diagnostic Workflow
Use the provided Node.js script to handle the complete workflow:

```bash
# Method 1: Using environment variables (recommended for persistence)
export DIAGNOSTIC_OSS_BUCKET=your-diagnostic-bucket
node scripts/ecs-diagnostic-cli.js \
  --region-id cn-hangzhou \
  --instance-ids i-xxxxxx

# Method 2: Using command-line arguments
node scripts/ecs-diagnostic-cli.js \
  --region-id cn-hangzhou \
  --instance-ids i-xxxxxx \
  --oss-bucket your-diagnostic-bucket
```

## Post-Diagnostic Analysis Procedure
After obtaining the diagnostic report (either from stdout in local mode or downloaded file in remote mode), follow these steps to analyze the results:

### Locate the Report
- **Local mode**: The JSON report is output directly to stdout. Redirect to a file if needed.
- **Remote mode**: The script outputs the path of the downloaded report file. Typical filename pattern is `diagnostic-report-YYYYMMDD-HHMMSS.json`.

### Extract and Analyze the "analysis" Section
Focus on the "analysis" part of the JSON report which contains diagnostic findings with the following structure:
- **name**: The diagnostic check identifier
- **level**: Severity level (info, warning, error)
- **params**: Additional parameters related to the finding
- **solution**: Link to documentation for resolving the issue

### Generate Human-Readable Report
Convert the analysis findings into a structured, human-readable format that includes:
- Clear categorization by severity level
- Natural language descriptions of each finding
- Recommended actions for warnings or errors
- Summary of system health status

### Interpret Common Findings
- **Info level**: Usually normal system configurations or status information
- **Warning level**: Potential issues that may require attention or optimization
- **Error level**: Critical issues that need immediate attention

## Reference: Node.js Script for Remote Mode

The `scripts/ecs-diagnostic-cli.js` script automates the complete remote diagnostic workflow. It is compiled from TypeScript source at `scripts/src/diagnostic-cli.ts` and runs on the same Node.js runtime as the agent CLI — no additional language runtime or package manager required.

**Important**: Always use this script as the primary method for remote diagnostics. If the script fails unexpectedly, refer to `references/manual-remote-procedure.md` for step-by-step manual procedures including troubleshooting guidance.

### Configuration Methods

The script supports two configuration methods for OSS settings with different regions and credentials:

#### Method 1: Environment Variables (Recommended)
Set environment variables for persistent configuration:

**Main ECS Configuration:**
```bash
export ALIBABA_CLOUD_ACCESS_KEY_ID=your_access_key_id
export ALIBABA_CLOUD_ACCESS_KEY_SECRET=your_access_key_secret
```

**OSS Configuration:**
```bash
export DIAGNOSTIC_OSS_BUCKET=your-diagnostic-bucket           # Required
export DIAGNOSTIC_OSS_PREFIX=your-diagnostic-prefix          # Optional, defaults to 'diagnostic-reports'
export DIAGNOSTIC_OSS_REGION=oss-bucket-region               # Optional, defaults to ECS region
export DIAGNOSTIC_OSS_ACCESS_KEY_ID=oss_access_key_id        # Optional, defaults to main credentials
export DIAGNOSTIC_OSS_ACCESS_KEY_SECRET=oss_access_key_secret # Optional, defaults to main credentials
```

Then run the script without specifying OSS settings:
```bash
node scripts/ecs-diagnostic-cli.js \
  --region-id cn-hangzhou \
  --instance-ids i-xxxxxx
```

#### Method 2: Command-Line Arguments
Specify OSS settings directly in the command (takes precedence over environment variables):
```bash
node scripts/ecs-diagnostic-cli.js \
  --region-id cn-hangzhou \
  --instance-ids i-xxxxxx \
  --oss-bucket your-diagnostic-bucket \
  --oss-prefix your-diagnostic-prefix \
  --oss-region oss-bucket-region \
  --oss-access-key-id oss_access_key_id \
  --oss-access-key-secret oss_access_key_secret
```

### Installation Requirements
The script is pre-compiled JavaScript. No additional installation steps are required beyond having Node.js available (same runtime as the agent CLI). All runtime dependencies (`@alicloud/ecs20140526`, `@alicloud/openapi-client`, `ali-oss`, etc.) are bundled as external modules that ship with the agent package.

### Usage Examples
```bash
# Basic usage with environment variables
node scripts/ecs-diagnostic-cli.js \
  --region-id cn-hangzhou \
  --instance-ids i-xxxxxx

# With command-line OSS bucket specification
node scripts/ecs-diagnostic-cli.js \
  --region-id cn-hangzhou \
  --instance-ids i-xxxxxx \
  --oss-bucket your-diagnostic-bucket

# With time range parameters
node scripts/ecs-diagnostic-cli.js \
  --region-id cn-hangzhou \
  --instance-ids i-xxxxxx \
  --since "2026-01-13T00:00:00Z" \
  --until "2026-01-14T00:00:00Z"
```

### Environment Variables
```bash
# Main credentials (required for ECS operations)
export ALIBABA_CLOUD_ACCESS_KEY_ID=your_access_key_id
export ALIBABA_CLOUD_ACCESS_KEY_SECRET=your_access_key_secret

# OSS configuration (DIAGNOSTIC_OSS_BUCKET is required)
export DIAGNOSTIC_OSS_BUCKET=your-diagnostic-bucket  # Required
export DIAGNOSTIC_OSS_PREFIX=your-diagnostic-prefix  # Optional, defaults to 'diagnostic-reports'
export DIAGNOSTIC_OSS_REGION=oss-bucket-region       # Optional, defaults to ECS region
```

### Command-Line Arguments
| Argument | Description |
|----------|-------------|
| `--region-id` | Alibaba Cloud region ID for ECS operations (required) |
| `--instance-ids` | Comma-separated list of ECS instance IDs (required) |
| `--oss-bucket` | OSS bucket name for storing diagnostic reports |
| `--oss-prefix` | Prefix for OSS object keys (default: 'diagnostic-reports') |
| `--oss-region` | Region for OSS bucket (default: ECS region) |
| `--since` | Start time for log analysis (ISO 8601 format) |
| `--until` | End time for log analysis (ISO 8601 format) |
| `--timeout` | Command execution timeout in seconds (default: 180) |
| `--expiration` | Pre-signed URL expiration in seconds (default: 3600) |
| `--verbose`, `-v` | Increase verbosity level (-v, -vv, -vvv) |
