import React from "react";
import { Text } from "ink";

/**
 * Persistent status bar showing key binding hints.
 * Rendered at the bottom of the app when InputPrompt is active.
 */
export function StatusBar(): React.ReactElement {
  return (
    <Text dimColor>
      {"  Ctrl+J newline │ ↑↓ history │ Enter submit │ /quit exit"}
    </Text>
  );
}
