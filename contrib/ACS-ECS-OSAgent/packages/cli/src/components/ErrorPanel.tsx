import React from "react";
import { Box, Text, useStdout } from "ink";

import { COLORS } from "../theme.js";

export type ErrorPanelProps = {
  /** Panel title (defaults to "Error") */
  title?: string;
  /** The error message to display */
  message: string;
  /** Optional actionable hint */
  hint?: string;
};

export function ErrorPanel({
  title = "Error",
  message,
  hint,
}: ErrorPanelProps): React.ReactElement {
  const { stdout } = useStdout();
  const cols = stdout.columns ?? 80;
  // Inner width: terminal columns - 2 border chars - 2 padding chars
  const innerWidth = Math.max(cols - 4, 30);

  const headerLabel = ` ❌ ${title} `;
  // Compute right-side horizontal rule length
  const rightDashCount = Math.max(innerWidth - headerLabel.length - 1, 3);
  const topBorder = `┌─${headerLabel}${"─".repeat(rightDashCount)}┐`;
  const bottomBorder = `└${"─".repeat(innerWidth + 2)}┘`;

  return (
    <Box flexDirection="column" marginTop={1}>
      <Text color={COLORS.error}>{topBorder}</Text>
      <Text color={COLORS.error}>
        {"│  "}
        <Text>{message}</Text>
      </Text>
      {hint && (
        <>
          <Text color={COLORS.error}>{"│"}</Text>
          <Text color={COLORS.error}>
            {"│  "}
            <Text dimColor>Hint: {hint}</Text>
          </Text>
        </>
      )}
      <Text color={COLORS.error}>{bottomBorder}</Text>
    </Box>
  );
}
