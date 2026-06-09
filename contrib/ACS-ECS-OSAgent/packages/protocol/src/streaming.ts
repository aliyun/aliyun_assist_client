/** Discriminated union for streaming deltas. */
export type ItemDelta =
  | AgentMessageDelta
  | ReasoningDelta
  | CommandOutputDelta;

export type AgentMessageDelta = {
  type: "agentMessage";
  delta: string;
};

export type ReasoningDelta = {
  type: "reasoning";
  delta: string;
};

export type CommandOutputDelta = {
  type: "commandOutput";
  delta: string;
  stream: "stdout" | "stderr";
};
