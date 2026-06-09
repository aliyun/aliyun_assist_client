import type { SkillService, SkillActivationResult } from "../SkillService.js";
import type { Tool, ToolExecutionResult } from "./types.js";

export type ActivateSkillArgs = {
  name: string;
};

export type ActivateSkillErrorType = "not_found" | "activation_failed";

export type ActivateSkillResult = ToolExecutionResult & {
  skillResult: SkillActivationResult | null;
  /** Present only when exitCode !== 0 */
  errorType?: ActivateSkillErrorType;
};

export interface ActivateSkillTool extends Tool<ActivateSkillArgs> {
  execute(args: ActivateSkillArgs, context: { cwd: string }): Promise<ActivateSkillResult>;
}

export function createActivateSkillTool(skillService: SkillService): ActivateSkillTool {
  const skillNames = skillService.getSkillNames();

  return {
    definition: {
      type: "function",
      function: {
        name: "activate_skill",
        description: skillService.generateToolDescription(),
        parameters: {
          type: "object",
          properties: {
            name: {
              type: "string",
              description: "The name of the skill to activate",
              ...(skillNames.length > 0 ? { enum: skillNames } : {}),
            },
          },
          required: ["name"],
        },
      },
    },

    async execute(args, _context): Promise<ActivateSkillResult> {
      const startTime = Date.now();

      try {
        const skillResult = await skillService.activateSkill(args.name);
        const durationMs = Date.now() - startTime;

        if (skillResult) {
          // Format result as XML
          const resourcesXml = skillResult.resources.length > 0
            ? `\n  <resources>\n${skillResult.resources.map(r => `    <file>${r}</file>`).join("\n")}\n  </resources>`
            : "";

          const output = `<skill name="${skillResult.name}">
  <basedir>${skillResult.basedir}</basedir>

${skillResult.content}${resourcesXml}
</skill>`;

          return {
            output,
            exitCode: 0,
            durationMs,
            skillResult,
          };
        } else {
          return {
            output: `Skill not found: ${args.name}`,
            exitCode: 1,
            durationMs,
            skillResult: null,
            errorType: "not_found",
          };
        }
      } catch (err) {
        // Convert activation errors (e.g., interpolation failures) to tool results
        // This allows the LLM to see the error and guide the user to fix it
        const durationMs = Date.now() - startTime;
        const errorMessage = err instanceof Error ? err.message : String(err);

        return {
          output: `Skill activation failed: ${errorMessage}`,
          exitCode: 1,
          durationMs,
          skillResult: null,
          errorType: "activation_failed",
        };
      }
    },
  };
}
