import React from "react";
import { Box } from "ink";

import { useMarkdown } from "../hooks/useMarkdown.js";

export type MessageStreamProps = {
  text: string;
};

/**
 * Renders accumulated streaming text from the assistant.
 * Phase 2: markdown rendering via marked AST → Ink components.
 */
export function MessageStream({ text }: MessageStreamProps): React.ReactElement {
  const rendered = useMarkdown(text);
  return <Box flexDirection="column">{rendered}</Box>;
}
