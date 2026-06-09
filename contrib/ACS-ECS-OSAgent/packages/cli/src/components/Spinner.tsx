import React, { useState, useEffect } from "react";
import { Text } from "ink";
import spinners from "cli-spinners";

import { COLORS } from "../theme.js";

// ---------------------------------------------------------------------------
// Hook — returns the current spinner frame character
// ---------------------------------------------------------------------------

export function useSpinnerFrame(): string {
  const spinner = spinners.dots;
  const [frameIndex, setFrameIndex] = useState(0);

  useEffect(() => {
    const timer = setInterval(() => {
      setFrameIndex((prev) => (prev + 1) % spinner.frames.length);
    }, spinner.interval);
    return () => clearInterval(timer);
  }, [spinner]);

  return spinner.frames[frameIndex] ?? "\u280B";
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export type SpinnerProps = {
  label: string;
};

export function Spinner({ label }: SpinnerProps): React.ReactElement {
  const frame = useSpinnerFrame();

  return (
    <Text>
      <Text color={COLORS.spinner}>{frame}</Text>
      <Text> {label}</Text>
    </Text>
  );
}
