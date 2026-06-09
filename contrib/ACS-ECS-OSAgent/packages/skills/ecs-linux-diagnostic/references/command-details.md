# ECS Linux Diagnostic Command Details

## Command Information
- **Name**: ACS-ECS-GuestOS-Diagnostic-for-linux.sh
- **Type**: Shell script
- **Provider**: AlibabaCloud.ECS.GuestOS
- **Purpose**: Run Linux baseline diagnostic for the instance

## Parameters

### Required Parameters
- `diagnostic_oss_url`: Diagnostic report upload URL; usually generated and filled in by the ECS instance health diagnostic service

### Optional Parameters
- `diagnostic_file_name`: (deprecated) File name of diagnostic report
- `ecsgohelper_interpargs`: Arguments for diagnostic tool runtime; not needed to fill but just use the default value
- `diagnostic_since`: Date and time that events should not be older than from diagnostic capabilities on logs; usually filled in by the ECS instance health diagnostic service
- `diagnostic_until`: Date and time that events should not be newer than from diagnostic capabilities on logs; usually filled in by the ECS instance health diagnostic service

## Command Execution Flow

1. Validate instance accessibility
2. Execute diagnostic checks on system components:
   - CPU and memory usage
   - File system health
   - Network configuration
   - System services status
   - Security configurations
3. Generate diagnostic report
4. Upload report to specified OSS location

## Typical Exit Codes

- **0**: Success - Diagnostic completed without critical issues
- **115**: Command execution failure (common for permission or connectivity issues)
- **Other non-zero**: Various diagnostic failures or warnings

## Expected Output Format

The diagnostic command generates a JSON report containing:
- System status information
- Component health checks
- Configuration details
- Potential issues or recommendations
- Resource usage statistics