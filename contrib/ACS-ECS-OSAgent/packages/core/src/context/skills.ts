import type { ContextSection } from "./types.js";

const SKILLS_CONTENT = `## Skills
A skill is a set of local instructions to follow that is stored in external \`SKILL.md\` file. For list of available skills or to read instructions of specific skill, refer to the \`activate_skill\` tool if available.

### How to use skills
- Discovery: Available skill list in this session is in the description of \`activate_skill\` tool.
- Trigger rules: If the user names a skill (with \`\$SkillName\` or plain text) OR the task clearly matches a skill's description shown above, you must use that skill for that turn. Multiple mentions mean use them all. Do not carry skills across turns unless re-mentioned.
- Missing/blocked: If a named skill isn't in the list or the path can't be read, say so briefly and continue with the best fallback.
- How to use a skill (progressive disclosure):
  1) After deciding to use a skill, call \`activate_skill\` tool with the skill's name to load its  full \`SKILL.md\` and other metadata.
  2) When \`SKILL.md\` references relative paths (e.g., \`scripts/foo.py\`), resolve them relative to the skill directory defined in metadata loaded, and only consider other paths if needed.
  3) If \`SKILL.md\` points to extra folders such as \`references/\`, load only the specific files needed for the request; don't bulk-load everything.
  4) If \`scripts/\` exist, prefer running or patching them instead of retyping large code blocks.
  5) If \`assets/\` or templates exist, reuse them instead of recreating from scratch.
- Coordination and sequencing:
  - If multiple skills apply, choose the minimal set that covers the request and state the order you'll use them.
  - Announce which skill(s) you're using and why (one short line). If you skip an obvious skill, say why.
- Context hygiene:
  - Keep context small: summarize long sections instead of pasting them; only load extra files when needed.
  - Avoid deep reference-chasing: prefer opening only files directly linked from \`SKILL.md\` unless you're blocked.
  - When variants exist (frameworks, providers, domains), pick only the relevant reference file(s) and note that choice.
- Safety and fallback: If a skill can't be applied cleanly (missing files, unclear instructions), state the issue, pick the next-best approach, and continue.`;

/** Instructions for the agent on how to discover and use skills. */
export const SKILLS_SECTION = {
  build() {
    return [{ role: "user" as const, content: SKILLS_CONTENT }];
  },
} satisfies ContextSection;
