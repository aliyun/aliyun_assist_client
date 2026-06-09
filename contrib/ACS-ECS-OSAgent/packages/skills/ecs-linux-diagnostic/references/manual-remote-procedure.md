# Manual Remote Diagnostic Procedure

Use this reference when the Node.js script (`scripts/ecs-diagnostic-cli.js`) fails unexpectedly. This documents the manual steps to perform remote diagnostics via Cloud Assistant and OSS.

## Command Reference

### Main Diagnostic Command
- **Name**: `ACS-ECS-GuestOS-Diagnostic-for-linux.sh`
- **Purpose**: Performs comprehensive baseline diagnostics on Linux instances
- **Parameters**:
  - `diagnostic_oss_url` (required): URL for uploading diagnostic reports
  - `diagnostic_since` (optional): Start time for log analysis
  - `diagnostic_until` (optional): End time for log analysis

## Complete Report Retrieval Process

When dealing with large diagnostic reports that exceed Cloud Assistant output limits (indicated by non-zero `Dropped` field):

### Step 1: Generate Pre-Signed URL
```bash
# Create a unique object key for this diagnostic run
OBJECT_KEY="diagnostic-reports/baseline-diagnostic-$(date +%Y%m%d-%H%M%S)-INSTANCE_ID.json"

# Generate pre-signed URL with 1-hour expiration
PRE_SIGNED_URL=$(aliyun oss presign --region cn-hangzhou oss://diagnostic-reports/$OBJECT_KEY --expires 3600)
```

### Step 2: Execute Diagnostic with Pre-Signed URL
```bash
aliyun ecs InvokeCommand \
  --InstanceId.1 INSTANCE_ID \
  --CommandId ACS-ECS-GuestOS-Diagnostic-for-linux.sh \
  --Parameters "{\"diagnostic_oss_url\":\"$PRE_SIGNED_URL\"}"
```

### Step 3: Wait for Completion and Retrieve Report
```bash
# Wait for command completion (poll until InvocationStatus is "Finished")
aliyun ecs DescribeInvocationResults --InvokeId INVOKE_ID

# Download the complete report from OSS
aliyun oss cp oss://diagnostic-reports/$OBJECT_KEY ./diagnostic-report.json
```

## Post-Validation Requirements

After invoking the diagnostic command, always validate the results:

### 1. Check Exit Code
- Exit code 0 indicates success
- Non-zero exit codes (like 115) indicate execution issues

### 2. Check Dropped Field
- If `Dropped` field is non-zero, output has been trimmed
- This indicates the diagnostic report was too large for the response
- Use the OSS approach described above for complete reports

### 3. Verify Invocation Status
- Check `InvocationStatus` field for "Finished" or "Failed"
- Look for `ErrorInfo` field for specific error details

### 4. Verify OSS Report Upload
- Check that the diagnostic report was successfully uploaded to OSS
- Validate the report file exists and is accessible

## Common Issues and Solutions

### Issue 1: Command Execution Failure
**Symptom**: Exit code 115 or similar error codes, `InvocationStatus` = "Failed"

**Solution**:
- Verify Cloud Assistant agent is running on the instance
- Check instance network connectivity to Alibaba Cloud services
- Ensure proper IAM permissions for ECS operations

### Issue 2: Invalid Instance ID
**Symptom**: "InstanceId is missing or empty" error

**Solution**:
- Use correct parameter format: `--InstanceId.1 instance-id`
- Verify instance exists and is in Running state

### Issue 3: Permission Denied
**Symptom**: Access denied errors when invoking commands

**Solution**:
- Ensure proper RAM role permissions for ECS operations
- Verify the account has necessary ECS privileges

### Issue 4: Truncated Output (Large Reports)
**Symptom**: Non-zero `Dropped` field in results or incomplete report

**Solution**:
- Use the pre-signed OSS URL approach to retrieve complete reports
- Generate unique object keys for each diagnostic run
- Verify the report was uploaded to OSS successfully

### Issue 5: OSS Upload Failure
**Symptom**: Diagnostic completes but no report in OSS

**Solution**:
- Verify OSS bucket exists and is accessible
- Check pre-signed URL hasn't expired
- Ensure instance has network access to OSS endpoint
- Verify OSS bucket permissions allow PUT operations

### Issue 6: Pre-Signed URL Expired
**Symptom**: OSS returns 403 or signature error

**Solution**:
- Generate a new pre-signed URL with longer expiration
- Default expiration is 3600 seconds (1 hour)
- For long-running diagnostics, use `--expires 7200` or higher

## Best Practices

1. **Always specify the correct region** when invoking commands
2. **Use unique OSS URLs** for each diagnostic run to prevent conflicts
3. **Check instance status** before running diagnostics
4. **Monitor command timeouts** (default 180 seconds for diagnostic commands)
5. **Review diagnostic reports** for actionable insights
6. **Perform post-validation** to ensure complete results
7. **Use pre-signed URLs** for large reports that exceed Cloud Assistant limits
8. **Validate OSS report uploads** to ensure complete data retrieval

## Sample Complete Workflow

```bash
# 1. Check instance status
INSTANCE_ID="i-xxxxxx"
REGION="cn-hangzhou"
aliyun ecs DescribeInstances --RegionId $REGION --InstanceIds "[\"$INSTANCE_ID\"]"

# 2. Generate pre-signed URL
OBJECT_KEY="diagnostic-reports/baseline-diagnostic-$(date +%Y%m%d-%H%M%S)-$INSTANCE_ID.json"
PRE_SIGNED_URL=$(aliyun oss presign --region $REGION oss://your-bucket/$OBJECT_KEY --expires 3600)

# 3. Execute diagnostic
INVOKE_RESULT=$(aliyun ecs InvokeCommand \
  --RegionId $REGION \
  --InstanceId.1 $INSTANCE_ID \
  --CommandId ACS-ECS-GuestOS-Diagnostic-for-linux.sh \
  --Parameters "{\"diagnostic_oss_url\":\"$PRE_SIGNED_URL\"}")

INVOKE_ID=$(echo $INVOKE_RESULT | jq -r '.InvokeId')

# 4. Poll for completion (simplified - in practice, loop with sleep)
aliyun ecs DescribeInvocationResults --RegionId $REGION --InvokeId $INVOKE_ID

# 5. Download report
aliyun oss cp oss://your-bucket/$OBJECT_KEY ./diagnostic-report.json

# 6. View analysis
cat diagnostic-report.json | jq '.analysis'
```
