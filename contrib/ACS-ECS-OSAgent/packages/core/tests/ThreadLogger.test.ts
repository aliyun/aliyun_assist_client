import { existsSync, readFileSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { ThreadLogger, type ThreadRecord } from "../src/ThreadLogger.js";

describe("ThreadLogger", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = mkdtempSync(join(tmpdir(), "thread-logger-"));
  });

  afterEach(() => {
    rmSync(tmpDir, { recursive: true, force: true });
  });

  it("creates the threads directory and a JSONL file named after the bare UUID", () => {
    const threadId = "thread_018f3c2b-7c5a-7b91-9c8e-2f1c4d3a9b12";
    const threadsDir = join(tmpDir, ".osagent", ".threads");

    const logger = new ThreadLogger(threadId, threadsDir);

    const expected = join(threadsDir, "018f3c2b-7c5a-7b91-9c8e-2f1c4d3a9b12.jsonl");
    expect(logger.path).toBe(expected);
    expect(existsSync(expected)).toBe(true);
    logger.close();
  });

  it("writes one JSON object per line with the documented envelope", () => {
    const threadId = "thread_abc";
    const logger = new ThreadLogger(threadId, tmpDir);
    logger.append("turn_started", { foo: 1 });
    logger.append("turn_completed", { bar: "x" });
    logger.close();

    const lines = readFileSync(logger.path, "utf8").trimEnd().split("\n");
    expect(lines).toHaveLength(2);

    const r1 = JSON.parse(lines[0]) as ThreadRecord;
    expect(r1.v).toBe(1);
    expect(r1.threadId).toBe(threadId);
    expect(r1.type).toBe("turn_started");
    expect(r1.data).toEqual({ foo: 1 });
    expect(typeof r1.ts).toBe("string");
    // ISO 8601 with millisecond precision
    expect(r1.ts).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);

    const r2 = JSON.parse(lines[1]) as ThreadRecord;
    expect(r2.type).toBe("turn_completed");
    expect(r2.data).toEqual({ bar: "x" });
  });

  it("appends to an existing file rather than truncating", () => {
    const logger1 = new ThreadLogger("thread_x", tmpDir);
    logger1.append("turn_started", {});
    logger1.close();

    const logger2 = new ThreadLogger("thread_x", tmpDir);
    logger2.append("turn_completed", {});
    logger2.close();

    const lines = readFileSync(logger1.path, "utf8").trimEnd().split("\n");
    expect(lines).toHaveLength(2);
  });

  it("after close(), append() is a no-op (does not throw, does not write)", () => {
    const logger = new ThreadLogger("thread_y", tmpDir);
    logger.append("turn_started", {});
    logger.close();

    // No throw; subsequent close() is also idempotent.
    expect(() => logger.append("turn_completed", {})).not.toThrow();
    expect(() => logger.close()).not.toThrow();

    const lines = readFileSync(logger.path, "utf8").trimEnd().split("\n");
    expect(lines).toHaveLength(1);
  });

  it("uses the threadId verbatim as filename when no `thread_` prefix is present", () => {
    const logger = new ThreadLogger("raw-id", tmpDir);
    expect(logger.path).toBe(join(tmpDir, "raw-id.jsonl"));
    logger.close();
  });
});
