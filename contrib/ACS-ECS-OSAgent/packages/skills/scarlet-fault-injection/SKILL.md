---
name: scarlet-fault-injection
description: Guide for injecting and revoking OS-level faults on Linux ECS instances using the ecsgo-scarlet chaos engineering tool (also invoked as the `scarlet` command). Use when the user needs to simulate OS faults such as high disk/memory/CPU usage, network packet drop, kernel panic, filesystem errors, SSH misconfigurations, etc. on Alibaba Cloud ECS instances, either locally or remotely via the cloud assistant plugin mechanism.
---

# Scarlet Fault Injection

`ecsgo-scarlet` (package name) / `scarlet` (command name) is a Linux chaos engineering toolbox for OS-level fault injection and revocation.

## Recommended Invocation on ECS: via `acs-plugin-manager`

In most cases, a properly installed Alibaba Cloud Assistant provides `acs-plugin-manager` as a symlink under a standard PATH directory (e.g. `/usr/sbin`). Invoke it by bare name directly:

```bash
# Inject a fault
acs-plugin-manager --exec --plugin ecsgo-scarlet --separator=' ' \
  --params="inject HighFilesystemInodeUsage --location /tmp --expected-percent 95"

# Revoke a fault
acs-plugin-manager --exec --plugin ecsgo-scarlet --separator=' ' \
  --params="revoke HighFilesystemInodeUsage --location /tmp"

# List all injectable faults
acs-plugin-manager --exec --plugin ecsgo-scarlet --separator=' ' --params="list"

# Inspect parameters before injecting
acs-plugin-manager --exec --plugin ecsgo-scarlet --separator=' ' \
  --params="inject HighFilesystemInodeUsage --help"
```

If `acs-plugin-manager` is **not found in PATH**, locate it using the search logic below:

```bash
search_acs_plugin_manager() {
    if [ -f "${1}/version" ]; then
        read current_version <"${1}/version"
        current_version_dir="${1}/$current_version"
        if [ -x "${current_version_dir}/acs-plugin-manager" ]; then
            echo "${current_version_dir}"
            return 0
        fi
    fi
    for i in ${1}/*.*.*.*; do
        if [ -x "${i}/acs-plugin-manager" ]; then
            echo "${i}"
            return 0
        fi
    done
    return 1
}

if [ -d /opt/local/share/aliyun-assist ]; then
    APM_DIR="$(search_acs_plugin_manager /opt/local/share/aliyun-assist)"
else
    APM_DIR="$(search_acs_plugin_manager /usr/local/share/aliyun-assist)"
fi
# Then invoke with full path:
$APM_DIR/acs-plugin-manager --exec --plugin ecsgo-scarlet --separator=' ' --params="..."
```

## Local Invocation (when `scarlet` is already installed)

```bash
# List all injectable faults
scarlet list

# Inspect inject parameters for a fault
scarlet inject <FaultName> --help

# Inspect revoke parameters for a fault
scarlet revoke <FaultName> --help

# Inject a fault
scarlet inject <FaultName> [fault-specific args]

# Revoke a fault
scarlet revoke <FaultName> [fault-specific args]

# Install dependencies of a fault (when required)
scarlet install_deps <FaultName>
```

## How to Inspect CLI Parameters

Always run `--help` before injecting to discover required and optional parameters:

```bash
scarlet inject HighFilesystemInodeUsage --help
scarlet revoke HighFilesystemInodeUsage --help
```

Parameters vary per fault — some have required args, some have optional ones with defaults. The `--help` output is the authoritative source.

## Examples

### HighFilesystemInodeUsage — High filesystem inode usage
- **inject**: `--location PATH` (required), `--expected-percent INTEGER` (default 99)
- **revoke**: `--location PATH` (required)

```bash
scarlet inject HighFilesystemInodeUsage --location /tmp --expected-percent 95
scarlet revoke HighFilesystemInodeUsage --location /tmp
```

### NetworkPacketDrop — Network packet drop
- **inject**: `--network-device NETWORK_DEVICE` (default eth0), `--dest-ip DEST_IP` (default 0.0.0.0)
- **revoke**: `--network-device NETWORK_DEVICE` (default eth0)

```bash
scarlet inject NetworkPacketDrop --network-device eth0 --dest-ip 10.0.0.1
scarlet revoke NetworkPacketDrop --network-device eth0
```

### NetworkTrafficBlocking — Block all network traffic (allows Alibaba Cloud management traffic)
- **inject/revoke**: `--backup-path PATH` (default /tmp/iptables.bak)

```bash
scarlet inject NetworkTrafficBlocking
scarlet revoke NetworkTrafficBlocking
```

### KernelPanic — Kernel panic
Supports three modes. Example for the simplest case (disabling kdump):

```bash
scarlet inject KernelPanic --without-kdump
# (system will reboot)
scarlet revoke KernelPanic
```

For kdump scenarios, run `scarlet install_deps KernelPanic` first and reboot before injecting.

### OutOfMemory — Out of memory
Requires `stress-ng` dependency installed first:

```bash
scarlet install_deps OutOfMemory
scarlet inject OutOfMemory
scarlet revoke OutOfMemory
```

## Finding the Right Fault

`references/known-faults.yaml` provides a quick-reference snapshot for fast lookup. However, it may be outdated or incomplete. **The authoritative source is always:**

```bash
# Locally
scarlet list

# Or via acs-plugin-manager on ECS
acs-plugin-manager --exec --plugin ecsgo-scarlet --separator=' ' --params="list"
```

If a fault you are looking for is not found in the YAML, always fall back to running `scarlet list` directly.

For the quick-reference snapshot, see [references/known-faults.yaml](references/known-faults.yaml).
