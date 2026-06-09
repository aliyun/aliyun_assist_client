---
name: aliyun-cli
description: Interact with Alibaba Cloud services using the aliyun CLI tool to invoke OpenAPI operations. Use when you need to query or manage Alibaba Cloud resources like ECS instances, disks, security groups, etc. Provides direct access to OpenAPI through the aliyun CLI with proper parameter formatting and region specification.
---

# Aliyun CLI Skill

This skill enables direct interaction with Alibaba Cloud services using the aliyun CLI tool to invoke OpenAPI operations. The skill emphasizes the introspection process to discover and use appropriate OpenAPI operations for various cloud resources.

## When to Use This Skill

Use this skill when you need to:
1. Discover available OpenAPI operations for Alibaba Cloud services
2. Query cloud resources (ECS instances, disks, security groups, etc.) with the appropriate parameters
3. Execute any Alibaba Cloud OpenAPI operation available through the CLI
4. Access cloud resource metadata and configuration details

## Prerequisites

Ensure the aliyun CLI is properly installed and configured:
- Install via: `npm install -g @alicloud/aliyun-cli` or equivalent
- Configure credentials: `aliyun configure`
- Verify installation: `aliyun --version`

## Core Command Structure

The basic pattern for aliyun CLI commands:

```bash
aliyun <product> --region <region_id> <ApiName> --parameter1 value1 --parameter2 value2 ...
```

Where:
- `<product>`: Service name (e.g., `ecs`, `vpc`, `slb`, `rds`)
- `<region_id>`: Alibaba Cloud region ID (e.g., `cn-hangzhou`, `us-west-1`)
- `<ApiName>`: Specific OpenAPI operation name
- `--parameters`: Service-specific parameters with proper value formatting

## OpenAPI Discovery Process (Introspection)

The key to using the aliyun CLI effectively is the introspection process to discover available operations and their parameters:

### 1. Discover Available Operations for a Service

List all available API operations for a specific service:

```bash
aliyun help ecs
```

This command returns all available operations for ECS, including:
- Operation names (e.g., `DescribeInstances`, `CreateInstance`, `DeleteInstance`)
- Chinese descriptions of what each operation does
- The API version for the service

### 2. Get Detailed Parameter Information for a Specific Operation

For any specific operation, get detailed information about required and optional parameters:

```bash
aliyun help ecs DescribeInstances
```

This command provides:
- All required parameters (marked as `Required`)
- All optional parameters (marked as `Optional`)
- Parameter types (String, Boolean, Integer, RepeatList, etc.)
- Valid values and ranges for parameters
- Parameter descriptions and usage notes
- Examples of parameter formatting

### 3. Use the Discovered Information to Build Commands

Once you've discovered the parameters using the help system, construct your actual command:

```bash
aliyun ecs --region cn-hangzhou DescribeInstances --InstanceIds '["i-xxxxxxxxx"]'
```

The general pattern is:
1. Use `aliyun help <service>` to see all available operations
2. Use `aliyun help <service> <operation>` to see parameter details
3. Construct your command using the discovered parameters

## Parameter Formatting Guidelines

Based on the introspection process, you'll encounter these common parameter types:

### JSON Arrays (RepeatList)
For parameters accepting multiple values, use JSON array format:
```bash
--InstanceIds '["i-xxx1", "i-xxx2"]'
--SecurityGroupIds '["sg-xxx1", "sg-xxx2"]'
```

### Boolean Values
Use lowercase boolean values:
```bash
--DryRun true
--IoOptimized true
```

### Dates and Times
For date-time parameters, use UTC format as specified:
```bash
--Filter.1.Value 2023-01-01T00:00Z
```

### Tag Parameters
For tagging operations, use the indexed parameter format:
```bash
--Tag.1.Key "Environment" --Tag.1.Value "Production"
--Tag.2.Key "Owner" --Tag.2.Value "TeamA"
```

## Generic Usage Pattern

The standardized approach for using this skill:

1. **Discover** what you need to do by identifying the appropriate service and operation:
   ```bash
   aliyun help <service>
   ```

2. **Learn** the parameters for the operation:
   ```bash
   aliyun help <service> <operation>
   ```

3. **Execute** the operation with appropriate parameters:
   ```bash
   aliyun <service> --region <region> <operation> --parameter1 value1 --parameter2 value2
   ```

## Example Discovery Process

Following the general workflow:

1. **Discover ECS operations:**
   ```bash
   aliyun help ecs
   ```
   Shows operations like `DescribeInstances`, `DescribeSecurityGroups`, etc.

2. **Learn DescribeInstances parameters:**
   ```bash
   aliyun help ecs DescribeInstances
   ```
   Shows that `--RegionId` is required, `--InstanceIds` is optional, etc.

3. **Execute with specific parameters:**
   ```bash
   aliyun ecs --region cn-hangzhou DescribeInstances --InstanceIds '["i-xxxxxxxxx"]'
   ```

## Other Service Examples

This discovery process applies to all Alibaba Cloud services:

- **VPC Service**: `aliyun help vpc`
- **SLB Service**: `aliyun help slb`
- **RDS Service**: `aliyun help rds`
- **OSS Service**: `aliyun help oss`

Each service has its own set of operations discoverable through the help system.

## Error Handling

Common errors and solutions:
1. **Operation not found**: Use introspection to discover correct operation name: `aliyun help <service>`
2. **Missing required parameters**: Use `aliyun help <service> <operation>` to identify required parameters
3. **Region not specified**: Always include `--region <region_id>` for resource-specific operations
4. **Invalid parameter format**: Check `aliyun help <service> <operation>` for correct formatting
5. **Permission denied**: Verify your access credentials and permissions

## Mandatory Parameters for Specific Operations

Certain OpenAPI operations require specific parameters to be set for optimal results:

### GetInstanceConsoleOutput Operation

To retrieve the serial port (console) log of an ECS instance, use the `GetInstanceConsoleOutput` operation. The `ConsoleOutput` field in the response is base64-encoded, so you need to decode it:

```bash
aliyun ecs GetInstanceConsoleOutput --region cn-hangzhou --InstanceId 'i-xxxxxxxxx' | jq -r '.ConsoleOutput' | base64 -d |tail -n 300
```

This command:
1. Calls `GetInstanceConsoleOutput` to fetch the instance's serial console output
2. Uses `jq` to extract the `ConsoleOutput` field
3. Pipes through `base64 -d` to decode the base64-encoded content into readable text

## Security Considerations

- Use IAM policies to grant minimal required permissions
- Store access credentials securely
- Regularly rotate access keys
- Use temporary credentials when possible
- Be careful with commands that modify resources