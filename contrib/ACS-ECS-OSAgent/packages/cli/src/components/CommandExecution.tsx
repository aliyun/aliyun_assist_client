import React from "react";
import { Box, Text, useInput } from "ink";

import { STATUS_COLORS, TOOL_ICONS } from "../theme.js";
import type { ApprovalDecision, ToolCallState, ToolCallStatus } from "../hooks/useTurn.js";
import { useSpinnerFrame } from "./Spinner.js";

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

export type CommandExecutionProps = {
  state: ToolCallState;
  onDecision?: (decision: ApprovalDecision) => void;
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getShellLabel(toolType: "local" | "remote", target?: string): string {
  if (toolType === "remote") {
    return `Shell (${target ?? "remote"})`;
  }
  return "Shell";
}

function formatDuration(ms: number | null | undefined): string {
  if (ms == null) return "?";
  return `${(ms / 1000).toFixed(1)}s`;
}

function getLineColor(status: ToolCallStatus): string {
  switch (status) {
    case "pending_approval":
      return STATUS_COLORS.running;
    case "running":
      return STATUS_COLORS.running;
    case "completed":
      return STATUS_COLORS.success;
    case "failed":
      return STATUS_COLORS.error;
    case "denied":
      return STATUS_COLORS.denied;
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * Unified command execution display. Handles all lifecycle phases:
 * pending_approval → running → completed / failed / denied.
 *
 * Single icon at column 0, full-line coloring by status.
 */
export function CommandExecution({
  state,
  onDecision,
}: CommandExecutionProps): React.ReactElement {
  const {
    toolType = "local",
    command = "(unknown)",
    target,
    status,
    output,
    exitCode,
    durationMs,
  } = state;

  const spinnerFrame = useSpinnerFrame();

  // Capture keyboard input only when awaiting approval
  useInput(
    (input) => {
      if (!onDecision) return;
      switch (input.toLowerCase()) {
        case "y":
          onDecision("approved");
          break;
        case "n":
          onDecision("denied");
          break;
        case "a":
          onDecision("always_approve");
          break;
        default:
          break;
      }
    },
    { isActive: status === "pending_approval" },
  );

  const shellLabel = getShellLabel(toolType, target);
  const lineColor = getLineColor(status);

  // Build the command preview (truncate if too long)
  const commandPreview = command.length > 60
    ? command.slice(0, 59) + "…"
    : command;

  // Determine column-0 icon
  let col0Icon: string;
  if (status === "running" || status === "pending_approval") {
    col0Icon = spinnerFrame;
  } else if (status === "denied") {
    col0Icon = TOOL_ICONS.denied;
  } else {
    col0Icon = TOOL_ICONS.shell;
  }

  // Build suffix
  let suffix = "";
  if (status === "completed") {
    suffix = ` \u2014 done (${formatDuration(durationMs)}) \u2714`;
  } else if (status === "failed") {
    const dur = formatDuration(durationMs);
    suffix = exitCode != null
      ? ` \u2014 exit ${exitCode} (${dur}) \u2718`
      : ` \u2014 failed (${dur}) \u2718`;
  } else if (status === "denied") {
    suffix = " \u2014 denied";
  } else if (status === "running" || status === "pending_approval") {
    suffix = "...";
  }

  return (
    <Box flexDirection="column" marginBottom={1}>
      {/* Status line — full-line coloring */}
      <Text color={lineColor}>
        {col0Icon} {shellLabel}: {commandPreview}{suffix}
      </Text>

      {/* Approval prompt */}
      {status === "pending_approval" && (
        <Text>
          {"   "}
          <Text color="cyan" bold>[y]</Text>
          <Text>es  </Text>
          <Text color="cyan" bold>[n]</Text>
          <Text>o  </Text>
          <Text color="cyan" bold>[a]</Text>
          <Text>lways</Text>
        </Text>
      )}

      {/* Output (4-space indent, not colored) */}
      {status === "completed" && output && (
        <Box paddingLeft={4}>
          <Text>{output}</Text>
        </Box>
      )}
      {status === "failed" && output && (
        <Box paddingLeft={4}>
          <Text>{output}</Text>
        </Box>
      )}
    </Box>
  );
}
