import { EcsRemoteCommandService, type EcsScriptType } from "../EcsRemoteCommandService.js";
import type { Tool, ToolExecutionResult } from "./types.js";

export type RemoteShellArgs = {
  command: string;
  instance_id: string;
  region_id: string;
  type: EcsScriptType;
};

export const remoteShellTool: Tool<RemoteShellArgs> = {
  definition: {
    type: "function",
    function: {
      name: "remote_shell",
      description:
        "Run a command on a remote Alibaba Cloud ECS instance via Cloud Assistant. " +
        "Choose the type matching the remote instance OS: RunShellScript for Linux, " +
        "RunBatScript or RunPowerShellScript for Windows.",
      parameters: {
        type: "object",
        properties: {
          command: {
            type: "string",
            description: "The command to execute on the remote instance",
          },
          instance_id: {
            type: "string",
            description: "The ECS instance ID (e.g., i-bp1xxxxxxxxxxxxxx)",
          },
          region_id: {
            type: "string",
            description: "The Alibaba Cloud region ID (e.g., cn-hangzhou, cn-beijing)",
          },
          type: {
            type: "string",
            enum: ["RunShellScript", "RunBatScript", "RunPowerShellScript"],
            description:
              "Script type matching the remote instance OS. " +
              "Use RunShellScript for Linux, RunBatScript for Windows CMD, " +
              "RunPowerShellScript for Windows PowerShell.",
          },
        },
        required: ["command", "instance_id", "region_id", "type"],
      },
    },
  },

  async execute(args, _context): Promise<ToolExecutionResult> {
    const accessKeyId = process.env.ALIBABA_CLOUD_ACCESS_KEY_ID;
    const accessKeySecret = process.env.ALIBABA_CLOUD_ACCESS_KEY_SECRET;

    if (!accessKeyId || !accessKeySecret) {
      return {
        output: "Missing Alibaba Cloud credentials. Please set ALIBABA_CLOUD_ACCESS_KEY_ID and ALIBABA_CLOUD_ACCESS_KEY_SECRET environment variables.",
        exitCode: 1,
        durationMs: 0,
      };
    }

    try {
      const ecsService = new EcsRemoteCommandService(accessKeyId, accessKeySecret, args.region_id);
      return await ecsService.executeCommand(args.instance_id, args.command, args.type);
    } catch (error) {
      return {
        output: `Remote command execution failed: ${error instanceof Error ? error.message : String(error)}`,
        exitCode: 1,
        durationMs: 0,
      };
    }
  },
};
