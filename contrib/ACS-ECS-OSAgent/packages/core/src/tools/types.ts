import type { ChatCompletionTool } from "openai/resources/chat/completions.mjs";

export type ToolExecutionResult = {
  output: string;
  exitCode: number;
  durationMs: number;
};

export type ToolExecutionContext = {
  cwd: string;
  // Future: profile, permissions, etc.
};

export interface Tool<TArgs = unknown> {
  readonly definition: ChatCompletionTool;
  execute(args: TArgs, context: ToolExecutionContext): Promise<ToolExecutionResult>;
}
