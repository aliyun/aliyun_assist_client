import type { ContextSection } from "./types.js";

/** System identity message for the OSAgent assistant. */
export const SYSTEM_SECTION = {
  build() {
    return [
      {
        role: "system" as const,
        content:
          "You are OSAgent, a helpful assistant for management and operations in operating systems.",
      },
    ];
  },
} satisfies ContextSection;
