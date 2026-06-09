import { existsSync, statSync } from "node:fs";
import { isAbsolute, resolve } from "node:path";
import { stdin, stdout } from "node:process";
import { createInterface } from "node:readline/promises";

import { program, type Command } from "commander";
import { AppServer, type AgentProfile, loadConfig, resolveModelConfig, type ResolvedModelConfig, type ResolvedConfig, formatTurnError } from "core";
import { render } from "ink";
import type {
  ClientResponse,
  CommandExecutionApprovalResponse,
  Notification,
  Request,
} from "protocol";
import React from "react";

import { makeAcpCommand } from "./acp.js";
import { App } from "./app.js";
import { runPlainTextInteractive } from "./plainTextMode.js";
import { VERSION } from "./version.js";

// ---------------------------------------------------------------------------
// Shared notification handler
// ---------------------------------------------------------------------------

function handleNotification(n: Notification): void {
  switch (n.method) {
    case "item/delta":
      process.stdout.write(n.params.delta.delta);
      break;
    case "item/started": {
      const item = n.params.item;
      if (item.type === "toolCall" && item.toolName === "activate_skill") {
        const args = JSON.parse(item.arguments) as { name?: string };
        process.stdout.write(`\n[Activating skill: ${args.name ?? "unknown"}...]\n`);
      } else if (item.type === "reasoning") {
        process.stdout.write(`\n[Thinking...]\n`);
      } else if (item.type === "mcpToolCall") {
        process.stdout.write(`\n[MCP: ${item.serverName}/${item.toolName} - Calling...]\n`);
      }
      break;
    }
    case "item/completed": {
      const item = n.params.item;
      if (item.type === "toolCall" && item.toolName === "activate_skill") {
        if (item.output) {
          if (item.status === "completed") {
            process.stdout.write(`[${item.output}]\n`);
          } else {
            // Show error message on activation failure
            process.stdout.write(`[Skill activation error: ${item.output}]\n`);
          }
        }
      } else if (item.type === "reasoning") {
        // Show reasoning summary if available
        if (item.content.length > 0) {
          const preview = item.content.substring(0, 100);
          process.stdout.write(`[Reasoning: ${preview}...]\n`);
        }
      } else if (item.type === "mcpToolCall") {
        if (item.status === "completed") {
          process.stdout.write(`[MCP: ${item.serverName}/${item.toolName} - Completed]\n`);
        } else if (item.status === "failed") {
          process.stdout.write(`[MCP: ${item.serverName}/${item.toolName} - Failed: ${item.output}]\n`);
        }
      }
      break;
    }
    case "turn/started":
    case "turn/completed":
    case "request/resolved":
    case "error":
      break;
    default:
      break;
  }
}

// ---------------------------------------------------------------------------
// Shared approval handler
// ---------------------------------------------------------------------------

async function handleApprovalRequest(
  req: Request,
  approvalMode: "prompt" | "yolo",
  rl: ReturnType<typeof createInterface>,
): Promise<CommandExecutionApprovalResponse> {
  if (req.method !== "approval/command_execution") {
    return { decision: "denied" };
  }

  const { command, cwd } = req.params;

  // YOLO mode: auto-approve all commands
  if (approvalMode === "yolo") {
    console.log(`\n[YOLO] Auto-approving: ${command ?? "(unknown)"}${cwd ? ` (in ${cwd})` : ""}`);
    return { decision: "approved" };
  }

  // Default: prompt user
  const prompt = `\nApprove command? ${command ?? "(unknown)"}${cwd ? ` (in ${cwd})` : ""}\n[y]es / [n]o: `;
  const answer = await rl.question(prompt);

  switch (answer.trim().toLowerCase()) {
    case "y":
    case "yes":
      return { decision: "approved" };
    default:
      return { decision: "denied" };
  }
}

// ---------------------------------------------------------------------------
// Run a single turn
// ---------------------------------------------------------------------------

async function runTurn(
  server: AppServer,
  threadId: string,
  input: string,
  approvalMode: "prompt" | "yolo",
  rl: ReturnType<typeof createInterface>,
): Promise<void> {
  const gen = server.turnStart({ threadId, input: [{ type: "text", text: input }] });
  let result = await gen.next();
  while (!result.done) {
    const msg = result.value;
    if ("id" in msg) {
      // Request — handle approval
      const request = msg as Request;
      const approvalDecision = await handleApprovalRequest(request, approvalMode, rl);
      const clientResponse: ClientResponse = { id: request.id, result: approvalDecision };
      result = await gen.next(clientResponse);
    } else {
      // Notification
      handleNotification(msg as Notification);
      result = await gen.next(undefined);
    }
  }
  process.stdout.write("\n");
}

// ---------------------------------------------------------------------------
// Interactive mode (Ink TUI)
// ---------------------------------------------------------------------------

async function runInteractive(server: AppServer, approvalMode: "prompt" | "yolo", cwd: string): Promise<void> {
  // Non-TTY fallback: use plain-text REPL when stdout is piped
  if (!process.stdout.isTTY) {
    await runPlainTextInteractive(server, approvalMode, cwd);
    return;
  }

  const { thread } = await server.threadStart({ cwd });

  // NOTE: Ink is intentionally rendered against the normal stdout buffer
  // (no alt-screen). A chat-style TUI must leave its history in terminal
  // scrollback after exit, so switching to the alternate screen buffer
  // (the `\x1b[?1049h` sequence used by editors / pagers) would be
  // user-hostile here. Ghost-line artefacts from streaming updates are
  // addressed by progressively promoting finalized turn segments into
  // Ink's <Static> area instead — see useTurn.ts.
  const instance = render(
    React.createElement(App, { server, threadId: thread.id, approvalMode }),
  );

  await instance.waitUntilExit();
}

// ---------------------------------------------------------------------------
// Non-interactive mode (exec subcommand)
// ---------------------------------------------------------------------------

async function readStdin(): Promise<string> {
  const chunks: string[] = [];
  for await (const chunk of stdin) {
    chunks.push(chunk);
  }
  return chunks.join("");
}

async function runNonInteractive(
  server: AppServer,
  promptArgs: string[],
  _options: { y?: boolean; yolo?: boolean; autoApproval?: string },
  cwd: string,
): Promise<void> {
  // In exec mode, YOLO is always enabled (default)
  // -y/--yolo flags are accepted but unnecessary
  const approvalMode = "yolo" as const;

  // Get prompt from args or stdin
  let prompt: string;

  if (promptArgs.length === 0 || (promptArgs.length === 1 && promptArgs[0] === "-")) {
    // Read from stdin
    prompt = await readStdin();
  } else {
    // Concatenate args with space
    prompt = promptArgs.join(" ");
  }

  if (!prompt.trim()) {
    throw new Error("No prompt provided");
  }

  const { thread } = await server.threadStart({ cwd });

  // Use a dummy readline interface for non-interactive mode (no actual prompting)
  const rl = createInterface({ input: stdin, output: stdout });
  try {
    await runTurn(server, thread.id, prompt, approvalMode, rl);
  } finally {
    rl.close();
  }
}

// ---------------------------------------------------------------------------
// CLI setup and entry point
// ---------------------------------------------------------------------------

// Determine approval mode from options for interactive mode
function getApprovalMode(opts: { y?: boolean; yolo?: boolean; autoApproval?: string }): "prompt" | "yolo" {
  if (opts.y || opts.yolo || opts.autoApproval === "yolo") {
    return "yolo";
  }
  return "prompt";
}

// Determine agent profile from CLI options, config file, or environment variable
function getProfile(opts: { profile?: string }, config: { profile: AgentProfile }): AgentProfile {
  const profile = opts.profile ?? config.profile;
  if (profile === "default" || profile === "leaf-node") {
    return profile;
  }
  throw new Error(`Invalid profile "${profile}". Valid values: default, leaf-node`);
}

// Resolve model configuration with graceful error handling
function getModelConfig(config: ResolvedConfig): ResolvedModelConfig {
  const result = resolveModelConfig(config);
  if (!result.ok) {
    throw new Error(result.error);
  }
  // Display deprecation warning when using env var fallback
  if (result.warning) {
    console.warn(`Warning: ${result.warning}\n`);
  }
  return result.config;
}

/**
 * Resolve working directory from accumulated -C option values.
 * Each -C path is resolved relative to the previous one (like git -C / make -C).
 * Empty string is a no-op. The final path must exist and be a directory.
 */
function resolveCwd(paths: string[] | undefined): string {
  if (!paths || paths.length === 0) {
    return process.cwd();
  }

  let cwd = process.cwd();
  for (const p of paths) {
    if (p === "") {
      continue; // empty -C "" is a no-op
    }
    cwd = isAbsolute(p) ? p : resolve(cwd, p);
  }

  if (!existsSync(cwd)) {
    throw new Error(`-C path does not exist: ${cwd}`);
  }
  if (!statSync(cwd).isDirectory()) {
    throw new Error(`-C path is not a directory: ${cwd}`);
  }
  return cwd;
}

async function main(): Promise<void> {
  program
    .name("osagent")
    .description("OS Agent CLI")
    .version(VERSION, "-v, --version", "Output the version number")
    .option("-C <path>", "Run as if started in <path> (repeatable, relative paths chain)", (val: string, prev: string[]) => [...prev, val], [] as string[])
    .option("-y, --yolo", "Auto-approve all tool calls (YOLO mode)")
    .option("--auto-approval <mode>", "Approval mode: 'yolo' for auto-approve, 'prompt' for interactive (default: prompt)")
    .option("--profile <profile>", "Agent profile: 'default' or 'leaf-node' (env: OSAGENT_PROFILE)")
    .action(async (options) => {
      const approvalMode = getApprovalMode(options);
      const cwd = resolveCwd(options.C as string[] | undefined);
      const config = loadConfig(cwd);
      const profile = getProfile(options, config);
      const modelConfig = getModelConfig(config);
      const server = new AppServer(config, modelConfig, profile);
      try {
        await runInteractive(server, approvalMode, cwd);
      } catch (error) {
        for (const line of formatTurnError(error)) {
          console.error(line);
        }
        process.exitCode = 1;
      } finally {
        try {
          await server.dispose();
        } catch {
          // Best-effort cleanup — disposal errors should not mask the original error
        }
      }
    });

  program
    .command("exec")
    .description("Execute a single prompt in non-interactive mode")
    .argument("[prompt...]", "Prompt text (reads from stdin if omitted or '-')")
    .option("-y, --yolo", "Auto-approve all tool calls (default in non-interactive mode)")
    .option("--auto-approval <mode>", "Approval mode (only 'yolo' supported in non-interactive mode)")
    .option("--profile <profile>", "Agent profile: 'default' or 'leaf-node' (env: OSAGENT_PROFILE)")
    .action(async function(this: Command, promptArgs, _options) {
      // Use optsWithGlobals() to get merged local and global option values
      const opts = this.optsWithGlobals();
      const cwd = resolveCwd(opts.C as string[] | undefined);
      const config = loadConfig(cwd);
      const profile = getProfile(opts, config);
      const modelConfig = getModelConfig(config);
      const server = new AppServer(config, modelConfig, profile);
      try {
        await runNonInteractive(server, promptArgs, opts, cwd);
      } catch (error) {
        for (const line of formatTurnError(error)) {
          console.error(line);
        }
        process.exitCode = 1;
      } finally {
        try {
          await server.dispose();
        } catch {
          // Best-effort cleanup — disposal errors should not mask the original error
        }
      }
    });

  program.addCommand(makeAcpCommand());

  await program.parseAsync();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
