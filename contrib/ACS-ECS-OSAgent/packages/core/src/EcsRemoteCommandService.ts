import ecs20140526_module, * as $Ecs20140526 from "@alicloud/ecs20140526";
import * as $OpenApi from "@alicloud/openapi-client";

const Ecs20140526 = ecs20140526_module.default;
const RunCommandRequest = $Ecs20140526.RunCommandRequest;
const DescribeInvocationResultsRequest = $Ecs20140526.DescribeInvocationResultsRequest;

export type EcsScriptType = "RunShellScript" | "RunBatScript" | "RunPowerShellScript";

export interface RemoteCommandResult {
  output: string;
  exitCode: number;
  durationMs: number;
  finishedTime?: string;
}

interface InvocationResult {
  instanceId: string;
  invokeRecordStatus: string;
  exitCode?: number;
  output?: string;
  finishedTime?: string;
}

export class EcsRemoteCommandService {
  // @ts-ignore -- Ecs20140526 type not available at compile time; resolved via Alibaba Cloud SDK at runtime
  private client: Ecs20140526;
  private regionId: string;

  constructor(accessKeyId: string, accessKeySecret: string, regionId: string) {
    const config = new $OpenApi.Config({
      accessKeyId,
      accessKeySecret,
      regionId,
    });
    this.client = new Ecs20140526(config);
    this.regionId = regionId;
  }

  async executeCommand(
    instanceId: string,
    command: string,
    type: EcsScriptType,
    timeoutSeconds: number = 180,
  ): Promise<RemoteCommandResult> {
    const startTime = Date.now();

    const invokeId = await this.runCommand(
      instanceId,
      command,
      type,
      timeoutSeconds,
    );

    if (!invokeId) {
      throw new Error("Failed to invoke remote command");
    }

    const timeoutMs = (timeoutSeconds + 30) * 1000;
    const result = await this.waitForResults(invokeId, timeoutMs);

    if (!result) {
      throw new Error("Failed to get command execution results");
    }

    const durationMs = Date.now() - startTime;

    return {
      output: result.output ?? "",
      exitCode: result.exitCode ?? -1,
      durationMs,
      finishedTime: result.finishedTime,
    };
  }

  private async runCommand(
    instanceId: string,
    command: string,
    type: EcsScriptType,
    timeoutSeconds: number,
  ): Promise<string | null> {
    const request = new RunCommandRequest({
      regionId: this.regionId,
      instanceId: [instanceId],
      type,
      commandContent: command,
      name: `remote-shell-${Date.now()}`,
      timeout: timeoutSeconds,
      keepCommand: false,
      contentEncoding: "PlainText",
    });

    try {
      const response = await this.client.runCommand(request);
      return response.body.invokeId ?? null;
    } catch (error) {
      throw new Error(
        `Failed to run command: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }

  private async waitForResults(
    invokeId: string,
    timeoutMs: number,
  ): Promise<InvocationResult | null> {
    const startTime = Date.now();
    let sleepInterval = 5000;

    while (Date.now() - startTime < timeoutMs) {
      const results = await this.fetchResults(invokeId);

      if (results && results.length > 0) {
        const result = results[0];

        if (result.invokeRecordStatus !== "Running") {
          return result;
        }
      }

      await this.sleep(sleepInterval);
      sleepInterval = Math.min(sleepInterval * 1.5, 30000);
    }

    return null;
  }

  private async fetchResults(
    invokeId: string,
  ): Promise<InvocationResult[] | null> {
    const request = new DescribeInvocationResultsRequest({
      regionId: this.regionId,
      invokeId,
    });

    try {
      const response = await this.client.describeInvocationResults(request);
      const invocation = response.body.invocation;

      if (!invocation?.invocationResults?.invocationResult) {
        return null;
      }

      return invocation.invocationResults.invocationResult.map(
        (r: {
          instanceId?: string;
          invokeRecordStatus?: string;
          exitCode?: number;
          output?: string;
          finishedTime?: string;
        }) => ({
          instanceId: r.instanceId ?? "",
          invokeRecordStatus: r.invokeRecordStatus ?? "Unknown",
          exitCode: r.exitCode,
          output: r.output ? this.decodeOutput(r.output) : undefined,
          finishedTime: r.finishedTime,
        }),
      );
    } catch {
      return null;
    }
  }

  private decodeOutput(output: string): string {
    try {
      return Buffer.from(output, "base64").toString("utf-8");
    } catch {
      return output;
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
