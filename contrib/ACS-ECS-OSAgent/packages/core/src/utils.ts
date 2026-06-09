import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Resolves the builtin skills directory bundled alongside this module.
 * When running from the CLI bundle, `dist/skills/` sits next to `dist/cli.js`.
 * Returns the absolute path if the directory exists, otherwise undefined.
 */
export function resolveBuiltinSkillsDir(): string | undefined {
  try {
    const thisFile = fileURLToPath(import.meta.url);
    const candidate = join(dirname(thisFile), "skills");
    return existsSync(candidate) ? candidate : undefined;
  } catch {
    return undefined;
  }
}
