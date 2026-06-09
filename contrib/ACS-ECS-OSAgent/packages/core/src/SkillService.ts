import { existsSync } from "node:fs";
import { readdir, readFile } from "node:fs/promises";
import { homedir } from "node:os";
import { join, dirname } from "node:path";

import * as yaml from "js-yaml";
import { z } from "zod";

import { McpServerConfigSchema, type InterpolationConfig } from "./config.js";
import { interpolateObject } from "./interpolation.js";

/**
 * Metadata extracted from SKILL.md frontmatter.
 */
export interface SkillMetadata {
  name: string;
  description: string;
}

/**
 * Zod schema for skill dependencies (agents/osagent.yaml).
 * Uses the same MCP server config schema as .osagent.yaml for consistency.
 */
const SkillDependenciesSchema = z.object({
  mcpServers: z.record(McpServerConfigSchema).optional(),
}).strict();

export type SkillDependencies = z.infer<typeof SkillDependenciesSchema>;

/**
 * Complete Zod schema for agents/osagent.yaml file.
 * References SkillDependenciesSchema for the dependencies section.
 */
const SkillMetadataFileSchema = z.object({
  dependencies: SkillDependenciesSchema.optional(),
}).strict();

export type SkillMetadataFile = z.infer<typeof SkillMetadataFileSchema>;

/**
 * Full skill data including body content and location.
 */
export interface Skill extends SkillMetadata {
  /** Absolute path to the SKILL.md file */
  location: string;
  /** Markdown body content (after frontmatter) */
  body: string;
}

/**
 * Result of activating a skill.
 */
export interface SkillActivationResult {
  name: string;
  basedir: string;
  content: string;
  resources: string[];
  /** MCP server dependencies to connect when skill is activated */
  dependencies?: SkillDependencies;
}

/**
 * Parses YAML frontmatter from SKILL.md content.
 * Returns { frontmatter, body } or null if parsing fails.
 */
function parseSkillMarkdown(content: string): { frontmatter: SkillMetadata; body: string } | null {
  // Match opening --- at start and closing ---
  const frontmatterMatch = content.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/);
  if (!frontmatterMatch) {
    return null;
  }

  const [, frontmatterStr, body] = frontmatterMatch;

  // Parse YAML frontmatter (simple key-value extraction)
  const lines = frontmatterStr.split("\n");
  const values: Record<string, string> = {};

  for (const line of lines) {
    const colonIndex = line.indexOf(":");
    if (colonIndex === -1) continue;

    const key = line.slice(0, colonIndex).trim();
    let value = line.slice(colonIndex + 1).trim();

    // Remove surrounding quotes if present
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }

    values[key] = value;
  }

  if (!values.name || !values.description) {
    return null;
  }

  return {
    frontmatter: {
      name: values.name,
      description: values.description,
    },
    body: body.trim(),
  };
}

/**
 * Recursively lists all files in a directory relative to the base directory.
 */
async function listFilesRecursive(baseDir: string, currentDir: string): Promise<string[]> {
  const files: string[] = [];
  const entries = await readdir(join(baseDir, currentDir), { withFileTypes: true });

  for (const entry of entries) {
    const relativePath = currentDir ? `${currentDir}/${entry.name}` : entry.name;

    if (entry.isDirectory()) {
      // Skip hidden directories and common non-skill directories
      if (entry.name.startsWith(".") || entry.name === "node_modules") {
        continue;
      }
      const subFiles = await listFilesRecursive(baseDir, relativePath);
      files.push(...subFiles);
    } else if (entry.isFile() && entry.name !== "SKILL.md") {
      files.push(relativePath);
    }
  }

  return files;
}

/**
 * Parses the skill dependencies YAML file (agents/osagent.yaml).
 * Returns SkillDependencies or null if file doesn't exist or is invalid.
 * 
 * Supports Handlebars interpolation with variables from the interpolation config.
 * Variables are referenced using {{ variable }} syntax in string values.
 * 
 * IMPORTANT: The file MUST be valid YAML conforming to the schema both BEFORE
 * and AFTER interpolation. Interpolation operates on the parsed object structure,
 * not raw text, ensuring structural integrity is preserved.
 * 
 * The format follows the same schema as .osagent.yaml configuration:
 * ```yaml
 * dependencies:
 *   mcpServers:
 *     serverName:
 *       type: stdio
 *       command: npx
 *       args: ["-y", "@modelcontextprotocol/server-filesystem"]
 *       env:
 *         TOKEN: "{{ access_token }}"
 *       allowedTools: ["read_file"]
 * ```
 * 
 * @param skillDir - Path to the skill directory
 * @param skillName - Name of the skill (for error messages and interpolation lookup)
 * @param interpolationConfig - Interpolation config from user/project configuration
 */
async function parseSkillDependenciesYaml(
  skillDir: string,
  skillName: string,
  interpolationConfig: InterpolationConfig
): Promise<SkillDependencies | null> {
  const depsPath = join(skillDir, "agents", "osagent.yaml");
  if (!existsSync(depsPath)) {
    return null;
  }

  // File exists - any validation failure from here should throw, not return null
  const rawContent = await readFile(depsPath, "utf-8");
  
  let raw: unknown;
  try {
    raw = yaml.load(rawContent);
  } catch (err) {
    throw new Error(
      `[skill:${skillName}] Failed to parse agents/osagent.yaml: ${err instanceof Error ? err.message : String(err)}`
    );
  }

  // PRE-INTERPOLATION VALIDATION:
  // The file MUST be valid YAML conforming to schema BEFORE interpolation.
  // This prevents arbitrary template manipulation that could twist file structure.
  // Note: Handlebars placeholders MUST be quoted in YAML (e.g., "{{ variable }}") to be parsed as strings.
  const preValidation = SkillMetadataFileSchema.safeParse(raw);
  if (!preValidation.success) {
    throw new Error(
      `[skill:${skillName}] Invalid agents/osagent.yaml (pre-interpolation): ${preValidation.error.message}. ` +
      `Hint: Ensure Handlebars placeholders are quoted (e.g., "{{ variable }}").`
    );
  }

  // Apply Handlebars interpolation on the parsed object (not raw text)
  const context = interpolationConfig[skillName] ?? {};
  const interpolated = interpolateObject(preValidation.data, context, skillName);

  // POST-INTERPOLATION VALIDATION:
  // Verify the interpolated object still conforms to schema.
  const postValidation = SkillMetadataFileSchema.safeParse(interpolated);
  if (!postValidation.success) {
    throw new Error(
      `[skill:${skillName}] Invalid agents/osagent.yaml (post-interpolation): ${postValidation.error.message}`
    );
  }

  // Return dependencies only if mcpServers is non-empty
  if (!postValidation.data.dependencies?.mcpServers ||
      Object.keys(postValidation.data.dependencies.mcpServers).length === 0) {
    return null;
  }

  return postValidation.data.dependencies;
}

/**
 * Service for discovering and loading Agent Skills.
 *
 * Skills are discovered from (in ascending priority order):
 * - `~/.agents/skills/` (user-level, cross-agent convention)
 * - `~/.osagent/skills/` (user-level, OSAgent-specific)
 * - `.agents/skills/` (project-level, cross-agent convention)
 * - `.osagent/skills/` (project-level, OSAgent-specific)
 *
 * Higher priority directories override skills with the same name from lower priority.
 */
export class SkillService {
  private skills = new Map<string, Skill>();
  private cwd: string;
  private builtinSkillsDir: string | undefined;
  private interpolationConfig: InterpolationConfig;

  constructor(
    cwd: string = process.cwd(),
    builtinSkillsDir?: string,
    interpolationConfig: InterpolationConfig = {}
  ) {
    this.cwd = cwd;
    this.builtinSkillsDir = builtinSkillsDir;
    this.interpolationConfig = interpolationConfig;
  }

  /**
   * Scan skill directories and load metadata.
   * Should be called at session start.
   *
   * Directories are scanned in ascending priority order.
   * Skills from higher-priority directories override those with the same name.
   */
  async discover(): Promise<void> {
    this.skills.clear();

    const home = homedir();

    // Skill directories in ascending priority order
    // (later entries override earlier ones with the same skill name)
    const skillDirs = [
      join(home, ".agents", "skills"),        // 1. User-level cross-agent (lowest)
      join(home, ".osagent", "skills"),       // 2. User-level OSAgent-specific
      join(this.cwd, ".agents", "skills"),    // 3. Project-level cross-agent
      join(this.cwd, ".osagent", "skills"),   // 4. Project-level OSAgent-specific (highest)
    ];

    // Builtin skills have lowest priority (before user-level)
    if (this.builtinSkillsDir) {
      skillDirs.unshift(this.builtinSkillsDir);
    }

    for (const skillDir of skillDirs) {
      if (!existsSync(skillDir)) continue;

      try {
        const entries = await readdir(skillDir, { withFileTypes: true });

        for (const entry of entries) {
          if (!entry.isDirectory()) continue;

          const skillPath = join(skillDir, entry.name);
          const skillMdPath = join(skillPath, "SKILL.md");

          if (!existsSync(skillMdPath)) continue;

          try {
            const content = await readFile(skillMdPath, "utf-8");
            const parsed = parseSkillMarkdown(content);

            if (!parsed) continue;

            // Higher priority directories override lower priority
            // (we iterate from lowest to highest, so just overwrite)
            this.skills.set(parsed.frontmatter.name, {
              ...parsed.frontmatter,
              location: skillMdPath,
              body: parsed.body,
            });
          } catch {
            // Skip malformed skill files
          }
        }
      } catch {
        // Skip directories that can't be read
      }
    }
  }

  /**
   * Get list of available skills with name and description.
   * Used to populate the activate_skill tool description.
   */
  getCatalog(): Array<{ name: string; description: string }> {
    return [...this.skills.values()].map((skill) => ({
      name: skill.name,
      description: skill.description,
    }));
  }

  /**
   * Get list of skill names for tool parameter enum.
   */
  getSkillNames(): string[] {
    return [...this.skills.keys()];
  }

  /**
   * Check if a skill exists.
   */
  hasSkill(name: string): boolean {
    return this.skills.has(name);
  }

  /**
   * Activate a skill by loading its full content and resources.
   * Returns the skill activation result or null if not found.
   */
  async activateSkill(name: string): Promise<SkillActivationResult | null> {
    const skill = this.skills.get(name);
    if (!skill) return null;

    const basedir = dirname(skill.location);
    let resources: string[] = [];

    try {
      resources = await listFilesRecursive(basedir, "");
    } catch {
      // If we can't list resources, return empty array
      resources = [];
    }

    // Load skill dependencies from agents/osagent.yaml with interpolation
    const dependencies = await parseSkillDependenciesYaml(
      basedir,
      skill.name,
      this.interpolationConfig
    );

    return {
      name: skill.name,
      basedir,
      content: skill.body,
      resources,
      dependencies: dependencies ?? undefined,
    };
  }

  /**
   * Generate the description for the activate_skill tool.
   * Includes the skill catalog as a bulleted list.
   */
  generateToolDescription(): string {
    const catalog = this.getCatalog();

    if (catalog.length === 0) {
      return "The following skills provide specialized instructions for specific tasks.\nWhen a task matches a skill's description, call this `activate_skill` tool with the skill's name to load its full instructions.\n\nNo skills are currently available.";
    }

    const skillList = catalog
      .map((s) => `* \`${s.name}\`: ${s.description}`)
      .join("\n");

    return `The following skills provide specialized instructions for specific tasks.
When a task matches a skill's description, call this \`activate_skill\` tool with the skill's name to load its full instructions.

${skillList}`;
  }
}
