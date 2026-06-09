/** Status shared across all item types. */
export type ItemStatus = "in_progress" | "completed" | "failed";

/**
 * Discriminated union of all protocol items.
 * Each variant carries a literal `type` field as discriminant.
 */
export type Item =
  | AgentMessageItem
  | ReasoningItem
  | CommandExecutionItem
  | ToolCallItem
  | McpToolCallItem;

export type AgentMessageItem = {
  type: "agentMessage";
  id: string;
  content: string;
  status: ItemStatus;
};

export type ReasoningItem = {
  type: "reasoning";
  id: string;
  content: string;
  status: ItemStatus;
};

export type CommandExecutionItem = {
  type: "commandExecution";
  id: string;
  command: string;
  cwd: string;
  shell: string;
  output: string;
  exitCode: number | null;
  durationMs: number | null;
  status: ItemStatus;
};

export type ToolCallItem = {
  type: "toolCall";
  id: string;
  toolName: string;
  arguments: string;
  output: string;
  status: ItemStatus;
};

export type McpToolCallItem = {
  type: "mcpToolCall";
  id: string;
  serverName: string;
  toolName: string;
  arguments: string;
  output: string;
  status: ItemStatus;
};
