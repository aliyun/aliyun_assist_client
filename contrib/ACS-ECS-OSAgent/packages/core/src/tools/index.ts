export * from "./types.js";
export { runShellTool, type RunShellArgs } from "./run-shell.js";
export { remoteShellTool, type RemoteShellArgs } from "./remote-shell.js";
export {
  createActivateSkillTool,
  type ActivateSkillArgs,
  type ActivateSkillResult,
  type ActivateSkillTool,
} from "./activate-skill.js";

// Import for use in GENERAL_TOOLS registry
import { remoteShellTool } from "./remote-shell.js";
import { runShellTool } from "./run-shell.js";

/**
 * Registry of General Tools that can be enabled/disabled via profile.
 * Framework Tools (activate_skill) are NOT included here - they are always enabled.
 */
export const GENERAL_TOOLS = {
  run_shell: runShellTool,
  remote_shell: remoteShellTool,
} as const;

export type GeneralToolName = keyof typeof GENERAL_TOOLS;

/**
 * Get General Tools filtered by allowed tool list.
 * Only tools listed in allowedTools are included.
 */
export function getGeneralTools(allowedTools: string[]): Array<typeof GENERAL_TOOLS[GeneralToolName]> {
  return Object.entries(GENERAL_TOOLS)
    .filter(([name]) => allowedTools.includes(name))
    .map(([, tool]) => tool);
}
