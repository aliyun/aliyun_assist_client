---
name: sersi-remediation
description: Remediate common guest OS issues on Alibaba Cloud ECS instances using the Sersi tool. Use when the user needs to repair or fix diagnostic issues identified by the ECS diagnostic service, such as SSH configuration problems, system misconfigurations, or other remediable OS-level issues.
---

# Sersi - Alibaba Cloud ECS Instance Remediation

Sersi is an OS remediation tool for Alibaba Cloud ECS instances. It diagnoses and repairs common guest OS issues based on diagnostic issue IDs from the ECS diagnostic service.

## Invocation

Sersi runs on Alibaba Cloud ECS instances via the Cloud Assistant plugin manager (`acs-plugin-manager`).

### Basic Command

```bash
acs-plugin-manager --exec --plugin sersi --separator=' ' --params="<args>"
```

If `acs-plugin-manager` is not found in PATH, locate it under:
- `/usr/local/share/aliyun-assist/<version>/acs-plugin-manager`
- `/opt/local/share/aliyun-assist/<version>/acs-plugin-manager`

## CLI Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `--target=instance` | Yes | Specifies the repair target type (must be `instance` for online instance repair) |
| `--issueid=XXX` | Yes | The diagnostic issue ID to remediate (see [remediable-issues.yaml](references/remediable-issues.yaml)) |
| `--dry-run` | No | Preview the remediation script without executing |
| `-y` / `--yes` | No | Auto-confirm and skip user prompts |
| `--debug` | No | Enable verbose debug logging |

## Usage Examples

### Repair an instance with a specific issue ID

```bash
acs-plugin-manager --exec --plugin sersi --separator=' ' --params="--target=instance --issueid=GuestOS.SSH.PasswordAuthenticationDisabled"
```

### Preview remediation (dry-run mode)

```bash
acs-plugin-manager --exec --plugin sersi --separator=' ' --params="--target=instance --issueid=GuestOS.SSH.PasswordAuthenticationDisabled --dry-run"
```

### Auto-confirm remediation without prompts

```bash
acs-plugin-manager --exec --plugin sersi --separator=' ' --params="--target=instance --issueid=GuestOS.SSH.PasswordAuthenticationDisabled -y"
```

### Enable debug logging

```bash
acs-plugin-manager --exec --plugin sersi --separator=' ' --params="--target=instance --issueid=GuestOS.SSH.PasswordAuthenticationDisabled --debug"
```

## References

- [Remediable Issues](references/remediable-issues.yaml) - Complete list of supported diagnostic issue IDs
