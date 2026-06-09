# Quick Start: ECS Linux Diagnostic

## Simple One-Liner Commands

### Check Instance Status
```bash
aliyun ecs DescribeInstances --InstanceIds '["INSTANCE_ID"]'
```

### Run Baseline Diagnostic
```bash
aliyun ecs InvokeCommand \
  --InstanceId.1 INSTANCE_ID \
  --CommandId ACS-ECS-GuestOS-Diagnostic-for-linux.sh \
  --Parameters '{"diagnostic_oss_url":"oss://diagnostic-reports/baseline-diagnostic-report.json"}'
```

### Check Results
```bash
aliyun ecs DescribeInvocationResults --InvokeId INVOKE_ID
```

## Quick Troubleshooting

### If Command Fails with Exit Code 115:
1. Verify Cloud Assistant agent is running:
   ```bash
   # Check agent status on instance
   systemctl status cloud-assistant
   ```
2. Ensure instance has network access to Alibaba Cloud services
3. Confirm IAM permissions for ECS operations

### If Instance Not Found:
1. Double-check instance ID format
2. Verify instance exists in the specified region
3. Confirm instance is in Running state

## Next Steps After Diagnosis

1. Review the diagnostic report in your OSS bucket
2. Address any critical issues identified
3. Schedule regular baseline diagnostics for ongoing monitoring
4. Set up alerting based on diagnostic findings