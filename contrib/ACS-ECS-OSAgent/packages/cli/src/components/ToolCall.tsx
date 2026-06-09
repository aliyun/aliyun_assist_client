import React from "react";
import { Box, Text } from "ink";

import { STATUS_COLORS, TOOL_ICONS } from "../theme.js";
import type { ToolCallStatus } from "../hooks/useTurn.js";
import { useSpinnerFrame } from "./Spinner.js";

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

export type ToolCallProps = {
  toolName: string;
  status: ToolCallStatus;
  itemType: "toolCall" | "mcpToolCall";
  arguments?: string;
  output?: string;
  durationMs?: number | null;
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
 * Displays a non-command tool call (MCP or generic) as a single status line
 * with optional indented output. Single icon at column 0, full-line coloring.
 */
export function ToolCall({
  toolName,
  status,
  itemType,
  output,
  durationMs,
}: ToolCallProps): React.ReactElement {
  const spinnerFrame = useSpinnerFrame();
  const lineColor = getLineColor(status);

  // Type icon for column 0 in terminal states
  const typeIcon = itemType === "mcpToolCall" ? TOOL_ICONS.mcp : TOOL_ICONS.shell;

  // Label depends on item type
  const label = itemType === "mcpToolCall"
    ? `MCP: ${toolName}`
    : toolName;

  // Determine column-0 icon
  let col0Icon: string;
  if (status === "running" || status === "pending_approval") {
    col0Icon = spinnerFrame;
  } else if (status === "denied") {
    col0Icon = TOOL_ICONS.denied;
  } else {
    col0Icon = typeIcon;
  }

  // Build suffix
  let suffix = "";
  if (status === "completed") {
    const dur = formatDuration(durationMs);
    suffix = ` \u2014 done (${dur}) \u2714`;
  } else if (status === "failed") {
    const dur = formatDuration(durationMs);
    suffix = ` \u2014 failed (${dur}) \u2718`;
  } else if (status === "denied") {
    suffix = " \u2014 denied";
  } else if (status === "running" || status === "pending_approval") {
    suffix = "...";
  }

  return (
    <Box flexDirection="column" marginBottom={1}>
      {/* Status line — full-line coloring */}
      <Text color={lineColor}>
        {col0Icon} {label}{suffix}
      </Text>

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
