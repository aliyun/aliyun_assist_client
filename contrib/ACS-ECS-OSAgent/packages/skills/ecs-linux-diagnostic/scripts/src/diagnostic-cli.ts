#!/usr/bin/env node
/**
 * ECS Baseline Diagnostic with Full Report Retrieval
 *
 * Executes Linux baseline diagnostics on ECS instances and retrieves
 * the complete diagnostic report via OSS, bypassing Cloud Assistant output size limitations.
 *
 * Environment Variables (Required):
 *   ALIBABA_CLOUD_ACCESS_KEY_ID       - Alibaba Cloud access key ID
 *   ALIBABA_CLOUD_ACCESS_KEY_SECRET   - Alibaba Cloud access key secret
 *   DIAGNOSTIC_OSS_BUCKET             - OSS bucket name for storing reports
 *
 * Environment Variables (Optional):
 *   ALIBABA_CLOUD_REGION_ID           - Default region for ECS operations
 *   DIAGNOSTIC_OSS_PREFIX             - OSS object key prefix (default: diagnostic-reports)
 *   DIAGNOSTIC_OSS_REGION             - OSS bucket region (defaults to ECS region)
 *   DIAGNOSTIC_OSS_ACCESS_KEY_ID      - Separate OSS access key (defaults to main credentials)
 *   DIAGNOSTIC_OSS_ACCESS_KEY_SECRET  - Separate OSS secret key (defaults to main credentials)
 */

import ecs20140526_module, * as $Ecs20140526 from "@alicloud/ecs20140526";
import * as $OpenApi from "@alicloud/openapi-client";
import OSS from "ali-oss";
import { createWriteStream } from "node:fs";
import { stat } from "node:fs/promises";
import { pipeline } from "node:stream/promises";

const Ecs20140526 = ecs20140526_module.default;
const InvokeCommandRequest = $Ecs20140526.InvokeCommandRequest;
const DescribeInstancesRequest = $Ecs20140526.DescribeInstancesRequest;
const DescribeInvocationResultsRequest =
  $Ecs20140526.DescribeInvocationResultsRequest;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface DiagnosticOptions {
  regionId: string;
  instanceIds: string[];
  ossBucket: string;
  ossPrefix: string;
  ossRegion: string;
  mainAccessKeyId: string;
  mainAccessKeySecret: string;
  ossAccessKeyId: string;
  ossAccessKeySecret: string;
  since?: string;
  until?: string;
  timeoutSeconds: number;
  expirationSeconds: number;
}

interface InvocationResult {
  instanceId: string;
  invocationStatus: string;
  exitCode?: number;
  errorInfo?: string;
  dropped?: number;
}

type EcsClient = InstanceType<typeof Ecs20140526>;

// ---------------------------------------------------------------------------
// Logger
// ---------------------------------------------------------------------------

let verbosity = 0;

const log = {
  debug: (msg: string): void => {
    if (verbosity >= 3) process.stderr.write(`[DEBUG] ${msg}\n`);
  },
  info: (msg: string): void => {
    if (verbosity >= 2) process.stderr.write(`[INFO]  ${msg}\n`);
  },
  warn: (msg: string): void => {
    if (verbosity >= 1) process.stderr.write(`[WARN]  ${msg}\n`);
  },
  error: (msg: string): void => {
    process.stderr.write(`[ERROR] ${msg}\n`);
  },
};

// ---------------------------------------------------------------------------
// ECS helpers
// ---------------------------------------------------------------------------

function createEcsClient(
  accessKeyId: string,
  accessKeySecret: string,
  regionId: string,
): EcsClient {
  const config = new $OpenApi.Config({ accessKeyId, accessKeySecret, regionId });
  return new Ecs20140526(config);
}

async function validateInstanceStatus(
  client: EcsClient,
  regionId: string,
  instanceIds: string[],
): Promise<void> {
  const request = new DescribeInstancesRequest({
    regionId,
    instanceIds: JSON.stringify(instanceIds),
  });

  const response = await client.describeInstances(request);
  const instances = response.body?.instances?.instance ?? [];

  for (const instance of instances) {
    if (instance.status !== "Running") {
      throw new Error(
        `Instance ${instance.instanceId} is not in Running state (status: ${instance.status})`,
      );
    }
  }

  log.info(`All ${instanceIds.length} instance(s) are in Running state.`);
}

async function invokeDiagnosticCommand(
  client: EcsClient,
  regionId: string,
  instanceIds: string[],
  presignedUrl: string,
  since: string | undefined,
  until: string | undefined,
): Promise<string> {
  const params: Record<string, string> = { diagnostic_oss_url: presignedUrl };
  if (since) params["diagnostic_since"] = since;
  if (until) params["diagnostic_until"] = until;

  const request = new InvokeCommandRequest({
    regionId,
    commandId: "ACS-ECS-GuestOS-Diagnostic-for-linux.sh",
    instanceId: instanceIds,
    parameters: params,
  });

  const response = await client.invokeCommand(request);
  const invokeId = response.body?.invokeId;

  if (!invokeId) {
    throw new Error("InvokeCommand returned no invokeId");
  }

  log.info(`Diagnostic command invoked with ID: ${invokeId}`);
  return invokeId;
}

async function waitForCompletion(
  client: EcsClient,
  regionId: string,
  invokeId: string,
  timeoutSeconds: number,
): Promise<InvocationResult[]> {
  const deadline = Date.now() + timeoutSeconds * 1000;
  const pollIntervalMs = 5000;

  log.info(
    `Waiting for invocation ${invokeId} to complete (timeout: ${timeoutSeconds}s)...`,
  );

  while (Date.now() < deadline) {
    const request = new DescribeInvocationResultsRequest({
      regionId,
      invokeId,
    });

    const response = await client.describeInvocationResults(request);
    const rawResults =
      response.body?.invocation?.invocationResults?.invocationResult ?? [];

    if (rawResults.length > 0) {
      const totalCount = rawResults.length;
      let finishedCount = 0;

      const results: InvocationResult[] = rawResults.map((r: $Ecs20140526.DescribeInvocationResultsResponseBodyInvocationInvocationResultsInvocationResult) => ({
        instanceId: r.instanceId ?? "",
        invocationStatus: r.invocationStatus ?? "Unknown",
        exitCode: r.exitCode,
        errorInfo: r.errorInfo,
        dropped: r.dropped,
      }));

      for (const r of results) {
        log.debug(`Instance ${r.instanceId} status: ${r.invocationStatus}`);
        if (r.invocationStatus === "Success" || r.invocationStatus === "Failed") {
          finishedCount++;
        }
      }

      if (finishedCount === totalCount) {
        log.info(`All ${totalCount} instance(s) finished.`);
        return results;
      }

      log.debug(`Progress: ${finishedCount}/${totalCount} finished. Waiting...`);
    }

    await new Promise<void>((resolve) => setTimeout(resolve, pollIntervalMs));
  }

  throw new Error(
    `Diagnostic command did not complete within ${timeoutSeconds}s timeout`,
  );
}

// ---------------------------------------------------------------------------
// OSS helpers
// ---------------------------------------------------------------------------

function createOssClient(
  region: string,
  bucket: string,
  accessKeyId: string,
  accessKeySecret: string,
): OSS {
  return new OSS({
    region: `oss-${region}`,
    bucket,
    accessKeyId,
    accessKeySecret,
  });
}

function generatePresignedUrl(
  ossClient: OSS,
  objectKey: string,
  expirationSeconds: number,
): string {
  return ossClient.signatureUrl(objectKey, {
    expires: expirationSeconds,
    method: "PUT",
  });
}

async function downloadReport(
  ossClient: OSS,
  objectKey: string,
  localPath: string,
): Promise<void> {
  const result = await ossClient.getStream(objectKey);
  const writeStream = createWriteStream(localPath);
  await pipeline(result.stream, writeStream);
}

// ---------------------------------------------------------------------------
// Main diagnostic workflow
// ---------------------------------------------------------------------------

async function runDiagnostic(opts: DiagnosticOptions): Promise<void> {
  // ECS always uses main credentials
  const ecsClient = createEcsClient(
    opts.mainAccessKeyId,
    opts.mainAccessKeySecret,
    opts.regionId,
  );

  // OSS may use separate credentials (fall back to main)
  const ossClient = createOssClient(
    opts.ossRegion,
    opts.ossBucket,
    opts.ossAccessKeyId,
    opts.ossAccessKeySecret,
  );

  // Build a timestamp string for object key and local file name
  const now = new Date();
  const timestamp = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, "0"),
    String(now.getDate()).padStart(2, "0"),
    "-",
    String(now.getHours()).padStart(2, "0"),
    String(now.getMinutes()).padStart(2, "0"),
    String(now.getSeconds()).padStart(2, "0"),
  ].join("");

  const objectKey = `${opts.ossPrefix}/baseline-diagnostic-${timestamp}-${opts.instanceIds.join("-")}.json`;

  // Step 1: Validate instances
  log.info("Validating instance status...");
  await validateInstanceStatus(ecsClient, opts.regionId, opts.instanceIds);

  // Step 2: Generate pre-signed OSS URL
  log.info(`Generating pre-signed URL for OSS object: ${objectKey}`);
  const presignedUrl = generatePresignedUrl(
    ossClient,
    objectKey,
    opts.expirationSeconds,
  );
  log.info("Pre-signed URL generated successfully.");

  // Step 3: Invoke diagnostic command
  log.info("Invoking diagnostic command...");
  const invokeId = await invokeDiagnosticCommand(
    ecsClient,
    opts.regionId,
    opts.instanceIds,
    presignedUrl,
    opts.since,
    opts.until,
  );

  // Step 4: Wait for completion
  log.info("Waiting for command completion...");
  const results = await waitForCompletion(
    ecsClient,
    opts.regionId,
    invokeId,
    opts.timeoutSeconds,
  );

  // Step 5: Print results summary to stderr
  process.stderr.write("\nDiagnostic Results Summary:\n");
  process.stderr.write(`${"-".repeat(40)}\n`);
  for (const r of results) {
    process.stderr.write(`Instance: ${r.instanceId}\n`);
    process.stderr.write(`  Status: ${r.invocationStatus}\n`);
    process.stderr.write(`  Exit Code: ${r.exitCode ?? "N/A"}\n`);
    process.stderr.write(`  Error Info: ${r.errorInfo ?? "None"}\n`);
    process.stderr.write(`  Dropped: ${r.dropped ?? 0}\n`);
    process.stderr.write("\n");
  }

  // Step 6: Download report from OSS
  const localReportPath = `./diagnostic-report-${timestamp}.json`;
  log.info(`Downloading complete report from OSS to ${localReportPath}...`);
  await downloadReport(ossClient, objectKey, localReportPath);

  const fileStats = await stat(localReportPath);
  log.info(`Report size: ${fileStats.size} bytes`);

  process.stderr.write("\nDiagnostic workflow completed successfully!\n");
  process.stderr.write(`Complete report saved to: ${localReportPath}\n`);

  // Output report path to stdout so callers can capture it
  process.stdout.write(`${localReportPath}\n`);
}

// ---------------------------------------------------------------------------
// CLI argument parsing
// ---------------------------------------------------------------------------

function parseArgs(argv: string[]): DiagnosticOptions {
  const args = argv.slice(2);

  const getFlag = (flag: string): string | undefined => {
    const idx = args.indexOf(flag);
    if (idx !== -1 && idx + 1 < args.length) return args[idx + 1];
    return undefined;
  };

  const hasFlag = (flag: string): boolean => args.includes(flag);

  if (hasFlag("--help") || hasFlag("-h")) {
    process.stdout.write(
      `Usage: ecs-diagnostic-cli [options]

Options:
  --region-id <id>              Alibaba Cloud region ID for ECS operations (required)
  --instance-ids <ids>          Comma-separated list of ECS instance IDs (required)
  --oss-bucket <name>           OSS bucket for diagnostic reports (required, or DIAGNOSTIC_OSS_BUCKET)
  --oss-prefix <prefix>         OSS object key prefix (default: diagnostic-reports)
  --oss-region <region>         OSS bucket region (default: ECS region)
  --oss-access-key-id <id>      OSS access key ID (default: main credentials)
  --oss-access-key-secret <s>   OSS access key secret (default: main credentials)
  --since <datetime>            Start time for log analysis (YYYY-mm-dd HH:MM:SS)
  --until <datetime>            End time for log analysis (YYYY-mm-dd HH:MM:SS)
  --timeout <seconds>           Command execution timeout in seconds (default: 180)
  --expiration <seconds>        Pre-signed URL expiration in seconds (default: 3600)
  --verbose, -v                 Increase verbosity level (repeat for more: -v -v -v)
  --help, -h                    Show this help message

Environment Variables:
  ALIBABA_CLOUD_ACCESS_KEY_ID       Main access key ID (required)
  ALIBABA_CLOUD_ACCESS_KEY_SECRET   Main access key secret (required)
  ALIBABA_CLOUD_REGION_ID           Default region ID
  DIAGNOSTIC_OSS_BUCKET             OSS bucket name (required if --oss-bucket not set)
  DIAGNOSTIC_OSS_PREFIX             OSS object key prefix
  DIAGNOSTIC_OSS_REGION             OSS bucket region
  DIAGNOSTIC_OSS_ACCESS_KEY_ID      Separate OSS access key
  DIAGNOSTIC_OSS_ACCESS_KEY_SECRET  Separate OSS secret key
`,
    );
    process.exit(0);
  }

  // Count --verbose / -v occurrences
  let verbosityLevel = 0;
  for (const arg of args) {
    if (arg === "--verbose" || arg === "-v") verbosityLevel++;
  }

  const mainAccessKeyId = process.env["ALIBABA_CLOUD_ACCESS_KEY_ID"] ?? "";
  const mainAccessKeySecret =
    process.env["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] ?? "";
  if (!mainAccessKeyId || !mainAccessKeySecret) {
    process.stderr.write(
      "Error: ALIBABA_CLOUD_ACCESS_KEY_ID and ALIBABA_CLOUD_ACCESS_KEY_SECRET must be set\n",
    );
    process.exit(1);
  }

  const regionId =
    getFlag("--region-id") ?? process.env["ALIBABA_CLOUD_REGION_ID"] ?? "";
  if (!regionId) {
    process.stderr.write("Error: --region-id is required\n");
    process.exit(1);
  }

  const instanceIdsRaw = getFlag("--instance-ids") ?? "";
  if (!instanceIdsRaw) {
    process.stderr.write("Error: --instance-ids is required\n");
    process.exit(1);
  }
  const instanceIds = instanceIdsRaw
    .split(",")
    .map((id) => id.trim())
    .filter((id) => id.length > 0);

  const ossBucket =
    getFlag("--oss-bucket") ?? process.env["DIAGNOSTIC_OSS_BUCKET"] ?? "";
  if (!ossBucket) {
    process.stderr.write(
      "Error: --oss-bucket or DIAGNOSTIC_OSS_BUCKET is required\n",
    );
    process.exit(1);
  }

  const ossPrefix =
    getFlag("--oss-prefix") ??
    process.env["DIAGNOSTIC_OSS_PREFIX"] ??
    "diagnostic-reports";
  const ossRegion =
    getFlag("--oss-region") ??
    process.env["DIAGNOSTIC_OSS_REGION"] ??
    regionId;
  const ossAccessKeyId =
    getFlag("--oss-access-key-id") ??
    process.env["DIAGNOSTIC_OSS_ACCESS_KEY_ID"] ??
    mainAccessKeyId;
  const ossAccessKeySecret =
    getFlag("--oss-access-key-secret") ??
    process.env["DIAGNOSTIC_OSS_ACCESS_KEY_SECRET"] ??
    mainAccessKeySecret;

  const timeoutSeconds = parseInt(getFlag("--timeout") ?? "180", 10);
  const expirationSeconds = parseInt(getFlag("--expiration") ?? "3600", 10);

  // Set module-level verbosity for the logger
  verbosity = verbosityLevel;

  return {
    regionId,
    instanceIds,
    ossBucket,
    ossPrefix,
    ossRegion,
    mainAccessKeyId,
    mainAccessKeySecret,
    ossAccessKeyId,
    ossAccessKeySecret,
    since: getFlag("--since"),
    until: getFlag("--until"),
    timeoutSeconds,
    expirationSeconds,
  } satisfies DiagnosticOptions;
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

(async () => {
  const opts = parseArgs(process.argv);

  try {
    await runDiagnostic(opts);
  } catch (error) {
    log.error(
      `Diagnostic failed: ${error instanceof Error ? error.message : String(error)}`,
    );
    process.exit(1);
  }
})();
