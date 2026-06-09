import { join } from "node:path";

import type { ChatCompletionTool } from "openai/resources/chat/completions";
import type {
  AgentLoop,
  AgentMessageItem,
  ClientResponse,
  CommandExecutionApprovalParams,
  CommandExecutionApprovalResponse,
  CommandExecutionItem,
  McpToolCallItem,
  Notification,
  ReasoningItem,
  Request,
  StopReason,
  ToolCallItem,
  Turn,
  TurnError,
  TurnStatus,
  UserInput as _ProtocolUserInput,
  ThreadStartParams as ProtocolThreadStartParams,
  ThreadStartResponse as ProtocolThreadStartResponse,
  TurnStartParams as ProtocolTurnStartParams,
} from "protocol";
import { uuidv7 } from "uuidv7";

import { type ResolvedConfig, type ResolvedModelConfig } from "./config.js";
import { Conversation } from "./Conversation.js";
import { classifyApiError, TurnFailedError } from "./errors.js";
import { McpService, type McpTool } from "./McpService.js";
import {
  type AgentProfile,
  type ProfileConfig,
  getProfileConfig,
} from "./profiles.js";
import { SkillService } from "./SkillService.js";
import { ThreadLogger } from "./ThreadLogger.js";
import {
  createActivateSkillTool,
  type Tool,
  type ActivateSkillTool,
  getGeneralTools,
} from "./tools/index.js";
import { resolveBuiltinSkillsDir } from "./utils.js";

export type { AgentProfile, ProfileConfig } from "./profiles.js";

// ---------------------------------------------------------------------------
// Local types — not part of the wire protocol
// ---------------------------------------------------------------------------

type UserInput = { type: string; text?: string; [key: string]: unknown };

type ThreadStatus =
  | { type: "idle" }
  | { type: "active"; activeFlags: string[] };

type Thread = {
  id: string;
  preview: string;
  ephemeral: boolean;
  modelProvider: string;
  createdAt: number;
  updatedAt: number;
  status: ThreadStatus;
  path: string | null;
  cwd: string;
  cliVersion: string;
  source: string;
  agentNickname: string | null;
  agentRole: string | null;
  gitInfo: null;
  name: string | null;
  turns: Turn[];
};

/**
 * Local params type — a superset of protocol's ThreadStartParams
 * with extra fields used by the Codex App Server wire format.
 */
type ThreadStartParams = ProtocolThreadStartParams & {
  cwd?: string | null;
  ephemeral?: boolean | null;
  modelProvider?: string | null;
  serviceTier?: string | null;
  approvalPolicy?: string | null;
};

/**
 * Local response type — a superset of protocol's ThreadStartResponse.
 * Includes `threadId` for AgentServer compatibility plus the full
 * Codex App Server response fields.
 */
type ThreadStartResponse = ProtocolThreadStartResponse & {
  thread: Thread;
  model: string;
  modelProvider: string;
  serviceTier: string | null;
  cwd: string;
  approvalPolicy: string;
  sandbox: { type: string };
  reasoningEffort: null;
};

type TurnStartParams = ProtocolTurnStartParams & {
  input: Array<UserInput>;
};

// ---------------------------------------------------------------------------

type ThreadState = {
  thread: Thread;
  conversation: Conversation;
  skillService: SkillService;
  mcpService: McpService;
  // null when thread.ephemeral === true; logging is skipped in that case.
  logger: ThreadLogger | null;
};

// Strip the API key before persisting model config to disk.
function redactModelConfig(modelConfig: ResolvedModelConfig): ResolvedModelConfig {
  return { ...modelConfig, apiKey: "***" };
}

export class AppServer {
  private threads = new Map<string, ThreadState>();
  private nextId = { thread: 0, turn: 0, item: 0, request: 0 };
  private profileConfig: ProfileConfig;
  private config: ResolvedConfig;
  private modelConfig: ResolvedModelConfig;

  constructor(config: ResolvedConfig, modelConfig: ResolvedModelConfig, profile?: AgentProfile) {
    this.config = config;
    this.modelConfig = modelConfig;
    const resolvedProfile = profile ?? config.profile;
    this.profileConfig = getProfileConfig(resolvedProfile);
  }

  private genId(kind: "thread" | "turn" | "item" | "request"): string {
    // Thread IDs are persisted as filenames and must remain unique across
    // process restarts and forks; use UUIDv7 so they are also time-sortable.
    // Other kinds remain process-local counters since they only need to be
    // unique within a thread's lifetime.
    if (kind === "thread") {
      return `thread_${uuidv7()}`;
    }
    return `${kind}_${++this.nextId[kind]}`;
  }

  async threadStart(params: ThreadStartParams): Promise<ThreadStartResponse> {
    const now = Math.floor(Date.now() / 1000);
    const threadId = this.genId("thread");
    const ephemeral = params.ephemeral ?? false;
    const cwd = params.cwd ?? process.cwd();

    // Open the JSONL log up front (unless the caller opted out via
    // `ephemeral: true`) so Thread.path can be populated atomically with
    // thread creation. The directory is created lazily by ThreadLogger.
    const logger = ephemeral ? null : new ThreadLogger(threadId, join(cwd, ".osagent", ".threads"));

    const thread: Thread = {
      id: threadId,
      preview: "",
      ephemeral,
      modelProvider: params.modelProvider ?? "openai",
      createdAt: now,
      updatedAt: now,
      status: { type: "idle" },
      path: logger?.path ?? null,
      cwd,
      cliVersion: "0.0.1",
      source: "cli",
      agentNickname: null,
      agentRole: null,
      gitInfo: null,
      name: null,
      turns: [],
    };

    const mcpService = new McpService();
    await mcpService.initialize(this.config.mcpServers);

    const conversation = new Conversation(this.modelConfig, this.config.proxy, thread.cwd);
    const skillService = new SkillService(
      thread.cwd,
      resolveBuiltinSkillsDir(),
      this.config.interpolation
    );
    await skillService.discover();
    this.threads.set(threadId, { thread, conversation, skillService, mcpService, logger });

    // First record: a snapshot of the Thread plus the resolved model config
    // (with the API key redacted) so a future replay tool has everything it
    // needs to reconstruct the runtime context.
    logger?.append("thread_started", {
      thread,
      modelConfig: redactModelConfig(this.modelConfig),
      profile: this.profileConfig,
    });

    return {
      threadId: threadId,
      thread,
      model: this.modelConfig.model,
      modelProvider: params.modelProvider ?? "openai",
      serviceTier: params.serviceTier ?? null,
      cwd: thread.cwd,
      approvalPolicy: params.approvalPolicy ?? "on-failure",
      sandbox: { type: "dangerFullAccess" },
      reasoningEffort: null,
    } satisfies ThreadStartResponse;
  }

  async *turnStart(params: TurnStartParams): AgentLoop {
    const state = this.threads.get(params.threadId);
    if (!state) {
      throw new Error(`Thread not found: ${params.threadId}`);
    }

    const { thread, conversation, skillService, mcpService, logger } = state;
    const turnId = this.genId("turn");

    // Build tools array with activate_skill using current skill catalog
    const activateSkillTool = createActivateSkillTool(skillService);
    const generalTools = getGeneralTools(this.profileConfig.allowedTools);
    const mcpTools = mcpService.getTools();
    const tools: Array<Tool | ActivateSkillTool | McpTool> = [...generalTools, activateSkillTool, ...mcpTools];

    // Extract text from UserInput[]
    const userText = params.input
      .filter(i => i.type === "text")
      .map(i => i.text ?? "")
      .join("\n");

    if (!thread.preview && userText) {
      thread.preview = userText.slice(0, 100);
    }

    const turn: Turn = {
      id: turnId,
      items: [],
      status: "in_progress",
      error: null,
    };

    // Thread becomes active
    thread.status = { type: "active", activeFlags: [] };
    thread.updatedAt = Math.floor(Date.now() / 1000);

    // 1. Turn started
    yield ({
      method: "turn/started",
      params: { threadId: thread.id, turnId },
    } satisfies Notification);
    logger?.append("turn_started", { turn, input: params.input });

    // Tool-calling loop
    let chatText: string | undefined = userText;
    const cancelled = false;

    try {
    while (!cancelled) {
      // -- Stream from Conversation --
      const agentItemId = this.genId("item");
      const reasoningItemId = this.genId("item");
      let agentStarted = false;
      let _reasoningStarted = false;
      let fullText = "";
      const reasoningContent: string[] = [];
      const toolCalls: Array<{ id: string; name: string; arguments: string }> = [];

      // Separate MCP tools from regular tools for OpenAI API
      const regularTools = tools.filter(t => !(t as McpTool).originalName) as Array<Tool | ActivateSkillTool>;
      const mcpToolsOnly = tools.filter(t => (t as McpTool).originalName) as McpTool[];

      // Convert MCP tool definitions to ChatCompletionTool format
      const mcpToolDefinitions: ChatCompletionTool[] = mcpToolsOnly.map(mcpTool => ({
        type: 'function' as const,
        function: {
          name: mcpTool.definition.name,
          description: mcpTool.definition.description,
          parameters: mcpTool.definition.input_schema as Record<string, unknown>,
        },
      }));

      const allToolDefinitions = [
        ...regularTools.map(t => t.definition),
        ...mcpToolDefinitions,
      ];

      logger?.append("llm_request", {
        turnId,
        toolNames: allToolDefinitions.map((t) => t.function.name),
        chatTextPresent: chatText !== undefined && chatText !== "",
      });

      for await (const event of conversation.chat(chatText, allToolDefinitions)) {
        if (event.type === "reasoningStarted") {
          // Emit item/started for reasoning
          yield ({
            method: "item/started",
            params: {
              item: {
                type: "reasoning",
                id: reasoningItemId,
                content: "",
                status: "in_progress",
              } satisfies ReasoningItem,
              threadId: thread.id,
              turnId,
            },
          } satisfies Notification);
          _reasoningStarted = true;
        } else if (event.type === "reasoning") {
          // Accumulate reasoning content
          reasoningContent.push(event.delta);
        } else if (event.type === "reasoningCompleted") {
          // Emit item/completed for reasoning
          yield ({
            method: "item/completed",
            params: {
              item: {
                type: "reasoning",
                id: reasoningItemId,
                content: reasoningContent.join(""),
                status: "completed",
              } satisfies ReasoningItem,
              threadId: thread.id,
              turnId,
            },
          } satisfies Notification);
          _reasoningStarted = false;
        } else if (event.type === "text") {
          if (!agentStarted) {
            yield ({
              method: "item/started",
              params: {
                item: {
                  type: "agentMessage",
                  id: agentItemId,
                  content: "",
                  status: "in_progress",
                } satisfies AgentMessageItem,
                threadId: thread.id,
                turnId,
              },
            } satisfies Notification);
            agentStarted = true;
          }
          fullText += event.delta;
          yield ({
            method: "item/delta",
            params: {
              threadId: thread.id,
              turnId,
              itemId: agentItemId,
              delta: { type: "agentMessage", delta: event.delta },
            },
          } satisfies Notification);
        } else {
          toolCalls.push(event.toolCall);
        }
      }

      // Complete agent message if started
      if (agentStarted) {
        yield ({
          method: "item/completed",
          params: {
            item: {
              type: "agentMessage",
              id: agentItemId,
              content: fullText,
              status: "completed",
            } satisfies AgentMessageItem,
            threadId: thread.id,
            turnId,
          },
        } satisfies Notification);
      }

      logger?.append("llm_response", {
        turnId,
        fullText,
        toolCalls: toolCalls.map((tc) => ({ id: tc.id, name: tc.name, arguments: tc.arguments })),
        reasoningContent,
      });

      // No tool calls — model is done
      if (toolCalls.length === 0) break;

      // -- Handle each tool call --
      for (const tc of toolCalls) {
        // Check if this is an MCP tool call (starts with 'mcp_' or 'skill_')
        if (tc.name.startsWith('mcp_') || tc.name.startsWith('skill_')) {
          // Look up the McpTool by mangledName to get the correct server
          const mcpTool = mcpToolsOnly.find(t => t.mangledName === tc.name);
          if (!mcpTool) {
            conversation.addToolResult(tc.id, `Unknown MCP tool: ${tc.name}`);
            continue;
          }

          const { server } = mcpTool;
          const mangledName = tc.name;
          const args = JSON.parse(tc.arguments);
          const mcpItemId = this.genId('item');

          // Emit item/started for mcpToolCall
          yield ({
            method: 'item/started',
            params: {
              item: {
                type: 'mcpToolCall',
                id: mcpItemId,
                serverName: server,
                toolName: mangledName,
                status: 'in_progress',
                arguments: JSON.stringify(args),
                output: '',
              } satisfies McpToolCallItem,
              threadId: thread.id,
              turnId,
            },
          } satisfies Notification);
          logger?.append("tool_started", { itemId: mcpItemId, server, tool: mangledName });

          try {
            const startTime = Date.now();
            const result = await mcpService.callTool(server, mangledName, args);
            const durationMs = Date.now() - startTime;

            // Emit item/completed
            yield ({
              method: 'item/completed',
              params: {
                item: {
                  type: 'mcpToolCall',
                  id: mcpItemId,
                  serverName: server,
                  toolName: mangledName,
                  status: 'completed',
                  arguments: JSON.stringify(args),
                  output: JSON.stringify(result),
                } satisfies McpToolCallItem,
                threadId: thread.id,
                turnId,
              },
            } satisfies Notification);
            logger?.append("tool_completed", { itemId: mcpItemId, durationMs });

            conversation.addToolResult(tc.id, JSON.stringify(result));
          } catch (error) {
            // Emit item/completed with failed status
            yield ({
              method: 'item/completed',
              params: {
                item: {
                  type: 'mcpToolCall',
                  id: mcpItemId,
                  serverName: server,
                  toolName: mangledName,
                  status: 'failed',
                  arguments: JSON.stringify(args),
                  output: error instanceof Error ? error.message : String(error),
                } satisfies McpToolCallItem,
                threadId: thread.id,
                turnId,
              },
            } satisfies Notification);
            logger?.append("tool_completed", { itemId: mcpItemId, error: String(error) });

            conversation.addToolResult(tc.id, `MCP tool call failed: ${error}`);
          }

          continue;
        }

        // Handle activate_skill separately (no approval needed)
        if (tc.name === "activate_skill") {
          const skillItemId = this.genId("item");
          const args = JSON.parse(tc.arguments) as { name: string };

          // Emit item/started for toolCall
          yield ({
            method: "item/started",
            params: {
              item: {
                type: "toolCall",
                id: skillItemId,
                toolName: "activate_skill",
                arguments: JSON.stringify(args),
                status: "in_progress",
                output: "",
              } satisfies ToolCallItem,
              threadId: thread.id,
              turnId,
            },
          } satisfies Notification);
          logger?.append("tool_started", { itemId: skillItemId, toolName: "activate_skill" });

          const result = await activateSkillTool.execute(args, { cwd: thread.cwd });
          const success = result.exitCode === 0;

          // If skill has MCP dependencies, connect them and update tools
          // Pass skillName to addServer for explicit tool naming: skill_<skill>_mcp_<server>_<tool>
          const mcpServersAdded: string[] = [];
          if (success && result.skillResult?.dependencies?.mcpServers) {
            const skillName = result.skillResult.name;
            const mcpServers = result.skillResult.dependencies.mcpServers;
            for (const [serverName, config] of Object.entries(mcpServers)) {
              const added = await mcpService.addServer(serverName, config, skillName);
              if (added) {
                // Internal server key is skill_server for tracking
                mcpServersAdded.push(`${skillName}_${serverName}`);
              }
            }

            // Update tools array with new MCP tools
            if (mcpServersAdded.length > 0) {
              const newMcpTools = mcpService.getTools().filter(
                t => mcpServersAdded.includes(t.server)
              );
              tools.push(...newMcpTools);
            }
          }

          // Build activation message
          // On failure, show the actual error message from result.output
          let activationText = success
            ? `Skill "${result.skillResult?.name}" activated.`
            : result.output;  // Show actual error (e.g., validation/interpolation failure)
          if (mcpServersAdded.length > 0) {
            activationText += ` Connected MCP servers: ${mcpServersAdded.join(", ")}.`;
          }

          // Emit item/completed
          yield ({
            method: "item/completed",
            params: {
              item: {
                type: "toolCall",
                id: skillItemId,
                toolName: "activate_skill",
                arguments: JSON.stringify(args),
                status: success ? "completed" : "failed",
                output: activationText,
              } satisfies ToolCallItem,
              threadId: thread.id,
              turnId,
            },
          } satisfies Notification);
          logger?.append("tool_completed", { itemId: skillItemId, success });

          conversation.addToolResult(tc.id, result.output);
          continue;
        }

        // Find the shell tool (exclude MCP tools)
        const nonMcpTools = tools.filter(t => !(t as McpTool).originalName) as Array<Tool | ActivateSkillTool>;
        const tool = nonMcpTools.find(t => t.definition.function.name === tc.name);
        if (!tool) {
          conversation.addToolResult(tc.id, `Unknown tool: ${tc.name}`);
          continue;
        }

        const cmdItemId = this.genId("item");
        const requestId = this.genId("request");

        // Build display command based on tool type
        const args = JSON.parse(tc.arguments);
        let displayCommand: string;
        if (tc.name === "remote_shell") {
          displayCommand = `[Remote: ${args.instance_id}@${args.region_id}] ${args.command}`;
        } else {
          displayCommand = args.command;
        }
        const shell = (args as { shell?: string }).shell ?? "";

        // item/started (commandExecution, in_progress)
        yield ({
          method: "item/started",
          params: {
            item: {
              type: "commandExecution",
              id: cmdItemId,
              command: displayCommand,
              cwd: thread.cwd,
              shell,
              status: "in_progress",
              output: "",
              exitCode: null,
              durationMs: null,
            } satisfies CommandExecutionItem,
            threadId: thread.id,
            turnId,
          },
        } satisfies Notification);
        logger?.append("tool_started", {
          turnId,
          itemId: cmdItemId,
          requestId,
          command: displayCommand,
          cwd: thread.cwd,
        });

        // Request: requestApproval — generator pauses here
        logger?.append("approval_requested", {
          turnId,
          itemId: cmdItemId,
          requestId,
          command: displayCommand,
          cwd: thread.cwd,
        });
        const response = (yield ({
          method: "approval/command_execution",
          id: requestId,
          params: {
            requestId,
            threadId: thread.id,
            turnId,
            itemId: cmdItemId,
            command: displayCommand,
            cwd: thread.cwd,
            shell,
          } satisfies CommandExecutionApprovalParams,
        } satisfies Request)) as unknown as ClientResponse | undefined;

        const responseResult = response?.result as CommandExecutionApprovalResponse | undefined;
        const decision = responseResult?.decision ?? "denied";
        logger?.append("approval_resolved", { requestId, decision });

        const resolution: "approved" | "denied" =
          (decision === "approved" || decision === "always_approve") ? "approved" : "denied";

        // request/resolved
        yield ({
          method: "request/resolved",
          params: { requestId, resolution },
        } satisfies Notification);

        if (decision === "approved" || decision === "always_approve") {
          // Execute the tool
          const result = await (tool as Tool | ActivateSkillTool).execute(args, { cwd: thread.cwd });

          if (result.output) {
            yield ({
              method: "item/delta",
              params: {
                threadId: thread.id,
                turnId,
                itemId: cmdItemId,
                delta: { type: "commandOutput", delta: result.output, stream: "stdout" },
              },
            } satisfies Notification);
          }

          yield ({
            method: "item/completed",
            params: {
              item: {
                type: "commandExecution",
                id: cmdItemId,
                command: displayCommand,
                cwd: thread.cwd,
                shell,
                status: "completed",
                output: result.output || "",
                exitCode: result.exitCode,
                durationMs: result.durationMs,
              } satisfies CommandExecutionItem,
              threadId: thread.id,
              turnId,
            },
          } satisfies Notification);
          logger?.append("tool_completed", {
            itemId: cmdItemId,
            exitCode: result.exitCode,
            durationMs: result.durationMs,
          });

          conversation.addToolResult(tc.id, result.output || "(no output)");
        } else {
          // denied — skip execution
          yield ({
            method: "item/completed",
            params: {
              item: {
                type: "commandExecution",
                id: cmdItemId,
                command: displayCommand,
                cwd: thread.cwd,
                shell,
                status: "failed",
                output: "",
                exitCode: null,
                durationMs: null,
              } satisfies CommandExecutionItem,
              threadId: thread.id,
              turnId,
            },
          } satisfies Notification);
          logger?.append("tool_completed", { itemId: cmdItemId, denied: true });

          conversation.addToolResult(tc.id, "Command declined by user.");
        }
      }

      // Next iteration: no user message, tool results already pushed
      chatText = undefined;
    }

    // Turn completed
    const completedStatus: TurnStatus = cancelled ? "cancelled" : "completed";
    const completedStopReason: StopReason = cancelled ? "cancelled" : "end_turn";
    turn.status = completedStatus;
    thread.status = { type: "idle" };
    thread.updatedAt = Math.floor(Date.now() / 1000);

    yield ({
      method: "turn/completed",
      params: {
        threadId: thread.id,
        turnId,
        status: completedStatus,
        stopReason: completedStopReason,
        error: null,
      },
    } satisfies Notification);
    logger?.append("turn_completed", { turn });
    } catch (error) {
      // Classify error into structured TurnError
      const turnError: TurnError = classifyApiError(error);
      turn.status = "failed";
      turn.error = turnError;
      logger?.append("turn_failed", { turn, error: turnError });
      throw new TurnFailedError(turnError);
    } finally {
      // Always reset thread to idle — even on error
      thread.status = { type: "idle" };
      thread.updatedAt = Math.floor(Date.now() / 1000);
    }
  }

  /**
   * Dispose of thread state and clean up resources.
   * Should be called when a thread is deleted or the server shuts down.
   */
  async disposeThread(threadId: string): Promise<void> {
    const state = this.threads.get(threadId);
    if (!state) return;

    state.logger?.close();
    await state.mcpService.shutdown();
    this.threads.delete(threadId);
  }

  /**
   * Dispose of all threads and clean up resources.
   * Should be called when the server shuts down.
   */
  async dispose(): Promise<void> {
    const threadIds = Array.from(this.threads.keys());
    await Promise.all(threadIds.map(id => this.disposeThread(id)));
  }
}
