import { execFile } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { promisify } from "node:util";

import { describe, it, expect } from "vitest";

const execFileAsync = promisify(execFile);

const CLI_PATH = join(import.meta.dirname, "..", "dist", "cli.js");

/**
 * E2E tests for the -C (working directory) CLI flag.
 *
 * These tests require:
 * - A built CLI bundle at dist/cli.js
 * - Valid LLM API credentials in environment (OPENAI_API_KEY or equivalent)
 *
 * The tests invoke the CLI binary in exec (non-interactive) mode with a real
 * LLM connection, verifying that the -C flag correctly propagates the working
 * directory to the agent's environment context.
 */
describe("-C flag E2E", () => {
  const timeout = 60_000; // LLM calls may take time

  it("propagates -C directory to LLM environment context", async () => {
    const tempDir = mkdtempSync(join(tmpdir(), "osagent-c-flag-"));

    // Ask the LLM to simply echo back the working directory from its context.
    // The environment prelude injects <currentWorkingDirectory> into the prompt.
    const { stdout } = await execFileAsync(
      process.execPath,
      [
        CLI_PATH,
        "-C", tempDir,
        "exec",
        "What is your current working directory? Reply with ONLY the absolute path, nothing else.",
      ],
      { timeout, env: { ...process.env, NODE_NO_WARNINGS: "1" } },
    );

    // The LLM should report the temp directory (from the environment prelude)
    expect(stdout).toContain(tempDir);
  }, timeout);

  it("errors on non-existent -C path", async () => {
    const bogusPath = "/nonexistent/path/that/does/not/exist";

    try {
      await execFileAsync(
        process.execPath,
        [CLI_PATH, "-C", bogusPath, "exec", "hello"],
        { timeout: 10_000, env: { ...process.env, NODE_NO_WARNINGS: "1" } },
      );
      expect.fail("Should have thrown");
    } catch (error) {
      const err = error as { stderr: string; code: number };
      expect(err.stderr).toContain("-C path does not exist");
    }
  });

  it("chains multiple -C paths correctly", async () => {
    // Create nested temp dirs: base/sub
    const baseDir = mkdtempSync(join(tmpdir(), "osagent-c-base-"));
    const { mkdirSync } = await import("node:fs");
    const subDir = join(baseDir, "sub");
    mkdirSync(subDir);

    // -C base -C sub should resolve to base/sub
    const { stdout } = await execFileAsync(
      process.execPath,
      [
        CLI_PATH,
        "-C", baseDir,
        "-C", "sub",
        "exec",
        "What is your current working directory? Reply with ONLY the absolute path, nothing else.",
      ],
      { timeout, env: { ...process.env, NODE_NO_WARNINGS: "1" } },
    );

    expect(stdout).toContain(subDir);
  }, timeout);

  it("-C with empty string is a no-op", async () => {
    const cwd = process.cwd();

    const { stdout } = await execFileAsync(
      process.execPath,
      [
        CLI_PATH,
        "-C", "",
        "exec",
        "What is your current working directory? Reply with ONLY the absolute path, nothing else.",
      ],
      { timeout, env: { ...process.env, NODE_NO_WARNINGS: "1" } },
    );

    // Should report the original cwd (or at least not crash)
    expect(stdout).toContain(cwd);
  }, timeout);
});
