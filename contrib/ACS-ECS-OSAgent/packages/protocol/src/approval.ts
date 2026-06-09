export type CommandExecutionApprovalParams = {
  requestId: string;
  threadId: string;
  turnId: string;
  itemId: string;
  command: string;
  cwd: string;
  shell: string;
};

export type CommandExecutionApprovalResponse = {
  decision: "approved" | "denied" | "always_approve";
};
