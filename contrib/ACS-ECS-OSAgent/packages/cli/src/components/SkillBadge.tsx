import React from "react";
import { Box, Text } from "ink";

import { STATUS_COLORS, TOOL_ICONS } from "../theme.js";
import { useSpinnerFrame } from "./Spinner.js";

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

export type SkillBadgeProps = {
  name: string;
  status: "activating" | "active" | "error";
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * Displays a lightweight single-line skill status indicator.
 * Single icon at column 0, full-line coloring by status.
 */
export function SkillBadge({ name, status }: SkillBadgeProps): React.ReactElement {
  const spinnerFrame = useSpinnerFrame();

  let lineColor: string;
  let col0Icon: string;
  let suffix: string;

  switch (status) {
    case "activating":
      lineColor = STATUS_COLORS.running;
      col0Icon = spinnerFrame;
      suffix = "...";
      break;
    case "active":
      lineColor = STATUS_COLORS.success;
      col0Icon = TOOL_ICONS.skill;
      suffix = " \u2014 active \u2714";
      break;
    case "error":
      lineColor = STATUS_COLORS.error;
      col0Icon = TOOL_ICONS.skill;
      suffix = " \u2014 failed \u2718";
      break;
  }

  return (
    <Box flexDirection="column" marginBottom={1}>
      <Text color={lineColor}>
        {col0Icon} Skill: {name}{suffix}
      </Text>
    </Box>
  );
}
