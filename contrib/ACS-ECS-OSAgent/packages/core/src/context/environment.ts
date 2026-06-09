import type { ContextSection } from "./types.js";

/** Create a dynamic environment information section with the given working directory. */
export function createEnvironmentSection(cwd: string): ContextSection {
  return {
    build() {
      const now = new Date();
      const dateStr = now.toLocaleDateString("en-US", {
        weekday: "long",
        year: "numeric",
        month: "long",
        day: "numeric",
      });
      const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

      const content = [
        "<currentWorkingDirectory>",
        cwd,
        "</currentWorkingDirectory>",
        "<currentDate>",
        dateStr,
        "</currentDate>",
        "<timezone>",
        timezone,
        "</timezone>",
      ].join("\n");

      return [{ role: "user" as const, content }];
    },
  };
}

/**
 * Default environment section using process.cwd().
 * @deprecated Prefer createEnvironmentSection(cwd) for explicit cwd propagation.
 */
export const ENVIRONMENT_SECTION = {
  build() {
    return createEnvironmentSection(process.cwd()).build();
  },
} satisfies ContextSection;
