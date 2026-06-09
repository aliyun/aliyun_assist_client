import { describe, it, expect } from "vitest";

import {
  SYSTEM_SECTION,
  SKILLS_SECTION,
  ENVIRONMENT_SECTION,
  createEnvironmentSection,
  type ContextSection,
} from "../src/context/index.js";

describe("SYSTEM_SECTION", () => {
  it("returns a single system-role message", () => {
    const msgs = SYSTEM_SECTION.build();
    expect(msgs).toHaveLength(1);
    expect(msgs[0]!.role).toBe("system");
  });

  it("contains OSAgent identity", () => {
    const msgs = SYSTEM_SECTION.build();
    expect((msgs[0] as { content: string }).content).toContain("OSAgent");
  });
});

describe("SKILLS_SECTION", () => {
  it("returns a single user-role message", () => {
    const msgs = SKILLS_SECTION.build();
    expect(msgs).toHaveLength(1);
    expect(msgs[0]!.role).toBe("user");
  });

  it("contains skill usage instructions", () => {
    const msgs = SKILLS_SECTION.build();
    expect((msgs[0] as { content: string }).content).toContain("activate_skill");
  });
});

describe("ENVIRONMENT_SECTION", () => {
  it("returns a single user-role message", () => {
    const msgs = ENVIRONMENT_SECTION.build();
    expect(msgs).toHaveLength(1);
    expect(msgs[0]!.role).toBe("user");
  });

  it("contains all XML tags", () => {
    const content = (ENVIRONMENT_SECTION.build()[0] as { content: string }).content;
    expect(content).toContain("<currentWorkingDirectory>");
    expect(content).toContain("</currentWorkingDirectory>");
    expect(content).toContain("<currentDate>");
    expect(content).toContain("</currentDate>");
    expect(content).toContain("<timezone>");
    expect(content).toContain("</timezone>");
  });

  it("populates dynamic values", () => {
    const content = (ENVIRONMENT_SECTION.build()[0] as { content: string }).content;
    expect(content).toContain(process.cwd());
    // Date and timezone strings are non-empty (sandwiched between tags)
    const dateMatch = content.match(/<currentDate>\n(.+)\n<\/currentDate>/);
    expect(dateMatch).not.toBeNull();
    expect(dateMatch![1]!.length).toBeGreaterThan(0);
    const tzMatch = content.match(/<timezone>\n(.+)\n<\/timezone>/);
    expect(tzMatch).not.toBeNull();
    expect(tzMatch![1]!.length).toBeGreaterThan(0);
  });
});

describe("createEnvironmentSection", () => {
  it("uses the provided cwd instead of process.cwd()", () => {
    const customCwd = "/custom/test/directory";
    const section = createEnvironmentSection(customCwd);
    const content = (section.build()[0] as { content: string }).content;
    expect(content).toContain(customCwd);
    expect(content).not.toContain(process.cwd());
  });

  it("returns a single user-role message", () => {
    const section = createEnvironmentSection("/tmp");
    const msgs = section.build();
    expect(msgs).toHaveLength(1);
    expect(msgs[0]!.role).toBe("user");
  });
});

describe("ContextSection composition", () => {
  it("produces 3 messages with correct roles when composed", () => {
    const sections: ContextSection[] = [SYSTEM_SECTION, SKILLS_SECTION, ENVIRONMENT_SECTION];
    const msgs = sections.flatMap((s) => s.build());
    expect(msgs).toHaveLength(3);
    expect(msgs[0]!.role).toBe("system");
    expect(msgs[1]!.role).toBe("user");
    expect(msgs[2]!.role).toBe("user");
  });
});
