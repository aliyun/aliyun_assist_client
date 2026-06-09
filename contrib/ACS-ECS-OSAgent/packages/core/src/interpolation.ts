import Handlebars from "handlebars";

export type InterpolationContext = Record<string, string>;

/**
 * Interpolate a single string value using Handlebars.
 *
 * @param template - String potentially containing {{ variable }} placeholders
 * @param context - Map of variable names to values
 * @param skillName - Skill name for error messages
 * @returns Interpolated string
 * @throws Error if a variable is referenced but not defined in context
 */
function interpolateString(
  template: string,
  context: InterpolationContext,
  skillName: string
): string {
  // Skip strings without Handlebars placeholders for performance
  if (!template.includes("{{")) {
    return template;
  }

  // Configure Handlebars to throw on missing variables
  const compiled = Handlebars.compile(template, { strict: true });
  try {
    return compiled(context);
  } catch (err) {
    if (err instanceof Error && err.message.includes("not defined")) {
      // Extract variable name from Handlebars error message
      const match = err.message.match(/"([^"]+)" not defined/);
      const varName = match?.[1] ?? "unknown";
      throw new Error(
        `[skill:${skillName}] Missing interpolation variable "${varName}". ` +
        `Define it in config under interpolation.${skillName}.${varName}`
      );
    }
    throw err;
  }
}

/**
 * Recursively walk an object and interpolate all string values.
 *
 * This operates on structured objects (after YAML parsing), ensuring that
 * the file structure is preserved and only string values are interpolated.
 *
 * @param obj - The object to interpolate (will be modified in place)
 * @param context - Map of variable names to values
 * @param skillName - Skill name for error messages
 * @returns The same object with interpolated string values
 * @throws Error if a variable is referenced but not defined in context
 */
export function interpolateObject<T>(obj: T, context: InterpolationContext, skillName: string): T {
  if (obj === null || obj === undefined) {
    return obj;
  }

  if (typeof obj === "string") {
    return interpolateString(obj, context, skillName) as T;
  }

  if (Array.isArray(obj)) {
    return obj.map(item => interpolateObject(item, context, skillName)) as T;
  }

  if (typeof obj === "object") {
    const result: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(obj)) {
      result[key] = interpolateObject(value, context, skillName);
    }
    return result as T;
  }

  // Primitives (number, boolean) pass through unchanged
  return obj;
}
