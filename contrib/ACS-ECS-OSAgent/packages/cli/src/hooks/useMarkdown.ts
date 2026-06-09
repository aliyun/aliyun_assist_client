import type { ReactNode } from "react";
import { useMemo } from "react";

import { renderMarkdown } from "../markdown/render.js";

/**
 * Memoised markdown renderer — re-parses only when the source text changes.
 */
export function useMarkdown(text: string): ReactNode {
  return useMemo(() => renderMarkdown(text), [text]);
}
