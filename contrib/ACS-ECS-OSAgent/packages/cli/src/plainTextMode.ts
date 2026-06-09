/**
 * Plain-text interactive REPL for non-TTY environments (piped output).
 * No colors, no spinners, no boxes — just raw text to stdout.
 */
import { stdin, stdout } from "node:process";
import { createInterface } from "node:readline/promises";

import type { AppServer } from "core";
import type {
  ClientResponse,
  CommandExecutionApprovalResponse,
  Notification,
  Request,
} from "protocol";

// ---------------------------------------------------------------------------
// Notification handler (plain text — no color, no decorations)
// ---------------------------------------------------------------------------

function handleNotification(n: Notification): void {
  switch (n.method) {
    case "item/delta":
      if (n.params.delta.type === "agentMessage") {
        stdout.write(n.params.delta.delta);
      }
      break;
    case "item/started": {
      const item = n.params.item;
      if (item.type === "toolCall" && item.toolName === "activate_skill") {
        const args = JSON.parse(item.arguments) as { name?: string };
        stdout.write(`\n[Activating skill: ${args.name ?? "unknown"}...]\n`);
      } else if (item.type === "reasoning") {
        stdout.write(`\n[Thinking...]\n`);
      } else if (item.type === "mcpToolCall") {
        stdout.write(`\n[MCP: ${item.serverName}/${item.toolName} - Calling...]\n`);
      } else if (item.type === "commandExecution") {
        stdout.write(`\n[Running: ${item.command}]\n`);
      }
      break;
    }
    case "item/completed": {
      const item = n.params.item;
      if (item.type === "toolCall" && item.toolName === "activate_skill") {
        if (item.output) {
          if (item.status === "completed") {
            stdout.write(`[${item.output}]\n`);
          } else {
            stdout.write(`[Skill activation error: ${item.output}]\n`);
          }
        }
      } else if (item.type === "reasoning") {
        if (item.content.length > 0) {
          const preview = item.content.substring(0, 100);
          stdout.write(`[Reasoning: ${preview}...]\n`);
        }
      } else if (item.type === "mcpToolCall") {
        if (item.status === "completed") {
          stdout.write(`[MCP: ${item.serverName}/${item.toolName} - Completed]\n`);
        } else if (item.status === "failed") {
          stdout.write(`[MCP: ${item.serverName}/${item.toolName} - Failed: ${item.output}]\n`);
        }
      } else if (item.type === "commandExecution") {
        if (item.output) {
          stdout.write(`${item.output}\n`);
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
// Run a single turn (plain text)
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
      const params = request.params as { command?: string; cwd?: string };

      let response: CommandExecutionApprovalResponse;

      if (approvalMode === "yolo") {
        stdout.write(`[Auto-approved: ${params.command ?? "(unknown)"}]\n`);
        response = { decision: "approved" };
      } else {
        const prompt = `\nApprove command? ${params.command ?? "(unknown)"}${params.cwd ? ` (in ${params.cwd})` : ""}\n[y]es / [n]o: `;
        const answer = await rl.question(prompt);
        const decision = answer.trim().toLowerCase();
        response = decision === "y" || decision === "yes"
          ? { decision: "approved" }
          : { decision: "denied" };
      }

      const clientResponse: ClientResponse = { id: request.id, result: response };
      result = await gen.next(clientResponse);
    } else {
      // Notification
      handleNotification(msg as Notification);
      result = await gen.next(undefined);
    }
  }

  stdout.write("\n");
}

// ---------------------------------------------------------------------------
// Plain text interactive REPL
// ---------------------------------------------------------------------------

export async function runPlainTextInteractive(
  server: AppServer,
  approvalMode: "prompt" | "yolo",
  cwd: string,
): Promise<void> {
  const { thread } = await server.threadStart({ cwd });
  const rl = createInterface({ input: stdin, output: stdout });

  try {
    while (true) {
      const input = await rl.question("> ");
      const trimmed = input.trim();

      if (trimmed === "/quit" || trimmed === "/exit" || trimmed === "exit") {
        break;
      }

      if (trimmed === "") {
        continue;
      }

      try {
        await runTurn(server, thread.id, trimmed, approvalMode, rl);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        stdout.write(`[Error] ${message}\n`);
      }
    }
  } finally {
    rl.close();
  }
}
