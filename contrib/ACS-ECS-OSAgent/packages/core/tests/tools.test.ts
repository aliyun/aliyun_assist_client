import { describe, it, expect } from "vitest";

import { getGeneralTools, GENERAL_TOOLS } from "../src/tools/index.js";

describe("GENERAL_TOOLS", () => {
  it("contains run_shell and remote_shell", () => {
    expect(GENERAL_TOOLS).toHaveProperty("run_shell");
    expect(GENERAL_TOOLS).toHaveProperty("remote_shell");
  });

  it("run_shell has correct tool definition name", () => {
    expect(GENERAL_TOOLS.run_shell.definition.function.name).toBe("run_shell");
  });

  it("remote_shell has correct tool definition name", () => {
    expect(GENERAL_TOOLS.remote_shell.definition.function.name).toBe("remote_shell");
  });
});

describe("getGeneralTools", () => {
  it("returns all tools when all are allowed", () => {
    const tools = getGeneralTools(["run_shell", "remote_shell"]);
    expect(tools).toHaveLength(2);
  });

  it("returns only run_shell when only run_shell is allowed", () => {
    const tools = getGeneralTools(["run_shell"]);
    expect(tools).toHaveLength(1);
    expect(tools[0].definition.function.name).toBe("run_shell");
  });

  it("returns only remote_shell when only remote_shell is allowed", () => {
    const tools = getGeneralTools(["remote_shell"]);
    expect(tools).toHaveLength(1);
    expect(tools[0].definition.function.name).toBe("remote_shell");
  });

  it("returns empty array when no tools are allowed", () => {
    const tools = getGeneralTools([]);
    expect(tools).toHaveLength(0);
  });

  it("ignores unknown tool names in allowed list", () => {
    const tools = getGeneralTools(["run_shell", "nonexistent_tool"]);
    expect(tools).toHaveLength(1);
    expect(tools[0].definition.function.name).toBe("run_shell");
  });
});
