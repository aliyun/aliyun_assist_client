export type AgentProfile = "default" | "leaf-node";

/**
 * Profile configuration interface.
 * Extensible for future settings beyond tool availability.
 */
export interface ProfileConfig {
  /** Profile name */
  name: AgentProfile;
  /** Human-readable description */
  description: string;
  /**
   * List of General Tool names that are allowed for the agent.
   * Only applies to General Tools (run_shell, remote_shell).
   * Framework Tools (activate_skill) are always enabled and not affected.
   */
  allowedTools: string[];
  /** Additional profile-specific settings (extensible) */
  settings: {
    /** Whether remote execution is allowed */
    allowRemoteExecution: boolean;
    // Future settings can be added here:
    // maxTokens?: number;
    // allowedSkills?: string[];
    // sandboxPolicy?: SandboxPolicy;
  };
}

/**
 * Built-in profile definitions.
 */
export const PROFILES: Record<AgentProfile, ProfileConfig> = {
  default: {
    name: "default",
    description: "No restrictions on the agent",
    allowedTools: ["run_shell", "remote_shell"],
    settings: {
      allowRemoteExecution: true,
    },
  },
  "leaf-node": {
    name: "leaf-node",
    description: "Leaf node mode - remote tools hidden",
    allowedTools: ["run_shell"],
    settings: {
      allowRemoteExecution: false,
    },
  },
};

/**
 * Get profile configuration by name.
 * Returns default profile if invalid name provided.
 */
export function getProfileConfig(profile: string): ProfileConfig {
  if (profile === "default" || profile === "leaf-node") {
    return PROFILES[profile];
  }
  return PROFILES.default;
}

/**
 * Check if a tool is available for the given profile.
 */
export function isToolAvailable(profile: AgentProfile, toolName: string): boolean {
  return PROFILES[profile].allowedTools.includes(toolName);
}
