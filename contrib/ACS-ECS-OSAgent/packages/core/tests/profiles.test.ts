import { describe, it, expect } from "vitest";

import {
  PROFILES,
  getProfileConfig,
  isToolAvailable,
} from "../src/profiles.js";

describe("PROFILES", () => {
  it("defines 'default' profile with both tools", () => {
    const p = PROFILES.default;
    expect(p.name).toBe("default");
    expect(p.allowedTools).toContain("run_shell");
    expect(p.allowedTools).toContain("remote_shell");
    expect(p.settings.allowRemoteExecution).toBe(true);
  });

  it("defines 'leaf-node' profile with only run_shell", () => {
    const p = PROFILES["leaf-node"];
    expect(p.name).toBe("leaf-node");
    expect(p.allowedTools).toContain("run_shell");
    expect(p.allowedTools).not.toContain("remote_shell");
    expect(p.settings.allowRemoteExecution).toBe(false);
  });
});

describe("getProfileConfig", () => {
  it("returns correct config for 'default'", () => {
    const config = getProfileConfig("default");
    expect(config).toBe(PROFILES.default);
  });

  it("returns correct config for 'leaf-node'", () => {
    const config = getProfileConfig("leaf-node");
    expect(config).toBe(PROFILES["leaf-node"]);
  });

  it("falls back to default for unknown profile name", () => {
    const config = getProfileConfig("nonexistent");
    expect(config).toBe(PROFILES.default);
  });

  it("falls back to default for empty string", () => {
    const config = getProfileConfig("");
    expect(config).toBe(PROFILES.default);
  });
});

describe("isToolAvailable", () => {
  it("returns true for run_shell in default profile", () => {
    expect(isToolAvailable("default", "run_shell")).toBe(true);
  });

  it("returns true for remote_shell in default profile", () => {
    expect(isToolAvailable("default", "remote_shell")).toBe(true);
  });

  it("returns true for run_shell in leaf-node profile", () => {
    expect(isToolAvailable("leaf-node", "run_shell")).toBe(true);
  });

  it("returns false for remote_shell in leaf-node profile", () => {
    expect(isToolAvailable("leaf-node", "remote_shell")).toBe(false);
  });

  it("returns false for unknown tool name", () => {
    expect(isToolAvailable("default", "unknown_tool")).toBe(false);
  });
});
