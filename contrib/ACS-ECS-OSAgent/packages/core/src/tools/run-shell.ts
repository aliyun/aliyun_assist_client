import { spawn } from "node:child_process";

import type { ChatCompletionTool } from "openai/resources/chat/completions.mjs";

import type { Tool } from "./types.js";

export type RunShellArgs = {
  command: string;
  shell?: string;
};

// ── Platform-specific schema ────────────────────────────────────────

interface PlatformSchemaPart {
  toolDescription: string;
  shellDescription: string;
  required: string[];
}

function platformSpecificSchemaPart(): PlatformSchemaPart {
  if (process.platform === "win32") {
    return {
      toolDescription:
        'Run a command in a shell. You must specify the shell: use "CMD" for cmd.exe, "PowerShell" for powershell.exe, or an absolute path to a custom shell.',
      shellDescription:
        'Shell to use. Use "CMD" for cmd.exe, "PowerShell" for powershell.exe, or an absolute path to a custom shell executable.',
      required: ["command", "shell"],
    };
  }
  return {
    toolDescription:
      "Run a command in a shell. The system-default shell is used unless overridden via the shell parameter.",
    shellDescription:
      "Absolute path to shell executable (e.g. /bin/bash, /bin/zsh). Omit to use the system-default shell.",
    required: ["command"],
  };
}

function buildRunShellDefinition(): ChatCompletionTool {
  const platform = platformSpecificSchemaPart();
  return {
    type: "function",
    function: {
      name: "run_shell",
      description: platform.toolDescription,
      parameters: {
        type: "object",
        properties: {
          command: {
            type: "string",
            description: "The shell command to execute",
          },
          shell: {
            type: "string",
            description: platform.shellDescription,
          },
        },
        required: platform.required,
      },
    },
  };
}

// ── Shell resolution ────────────────────────────────────────────────

function resolveShell(shell: string | undefined): true | string {
  if (shell === undefined) return true; // OS default
  if (process.platform === "win32") {
    if (shell === "CMD") return process.env.ComSpec ?? "cmd.exe";
    if (shell === "PowerShell") return "powershell.exe";
  }
  return shell; // Absolute path passthrough
}

// ── Tool export ─────────────────────────────────────────────────────

export const runShellTool: Tool<RunShellArgs> = {
  definition: buildRunShellDefinition(),

  execute(args, context): Promise<{ output: string; exitCode: number; durationMs: number }> {
    return new Promise((resolve) => {
      const start = Date.now();
      const proc = spawn(args.command, {
        shell: resolveShell(args.shell),
        cwd: context.cwd,
        stdio: ["ignore", "pipe", "pipe"],
      });

      let output = "";
      proc.stdout.on("data", (data: Buffer) => {
        output += data.toString();
      });
      proc.stderr.on("data", (data: Buffer) => {
        output += data.toString();
      });
      proc.on("close", (code) => {
        resolve({
          output,
          exitCode: code ?? 1,
          durationMs: Date.now() - start,
        });
      });
    });
  },
};
