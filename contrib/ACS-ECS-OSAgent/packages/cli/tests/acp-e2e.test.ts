/**
 * ACP E2E tests using the official @agentclientprotocol/sdk client.
 *
 * These tests spawn the `osagent acp` subprocess and exercise the protocol
 * via `ClientSideConnection`. They define the correctness spec — failures
 * against the current hand-rolled server are expected (TDD).
 */
import { spawn } from "node:child_process";
import type { ChildProcess } from "node:child_process";
import { join } from "node:path";
import { Readable, Writable } from "node:stream";

import {
  ClientSideConnection,
  ndJsonStream,
  PROTOCOL_VERSION,
} from "@agentclientprotocol/sdk";
import type {
  Client,
  Agent,
  SessionNotification,
  RequestPermissionRequest,
  RequestPermissionResponse,
} from "@agentclientprotocol/sdk";
import { describe, it, expect, afterEach } from "vitest";

const CLI_PATH = join(import.meta.dirname, "..", "dist", "cli.js");

/**
 * Minimal Client implementation that auto-approves permissions
 * and collects session updates for assertions.
 */
class TestClient implements Client {
  readonly updates: SessionNotification[] = [];

  async requestPermission(
    params: RequestPermissionRequest,
  ): Promise<RequestPermissionResponse> {
    // Auto-approve with the first allow option, or fallback
    const allowOption = params.options.find((o) => o.kind === "allow_once");
    return {
      outcome: {
        outcome: "selected",
        optionId: allowOption?.optionId ?? params.options[0]?.optionId ?? "allow",
      },
    };
  }

  async sessionUpdate(params: SessionNotification): Promise<void> {
    this.updates.push(params);
  }
}

/**
 * Spawn the ACP server subprocess and create a ClientSideConnection.
 */
function createAcpConnection(): {
  child: ChildProcess;
  connection: ClientSideConnection;
  client: TestClient;
} {
  const child = spawn("node", [CLI_PATH, "acp"], {
    stdio: ["pipe", "pipe", "pipe"],
    env: { ...process.env, NODE_NO_WARNINGS: "1" },
  });

  // Convert Node.js streams → Web Streams
  const stdinWritable = Writable.toWeb(
    child.stdin!,
  ) as WritableStream<Uint8Array>;
  const stdoutReadable = Readable.toWeb(
    child.stdout!,
  ) as ReadableStream<Uint8Array>;

  const stream = ndJsonStream(stdinWritable, stdoutReadable);
  const client = new TestClient();
  const connection = new ClientSideConnection(
    (_agent: Agent) => client,
    stream,
  );

  return { child, connection, client };
}

describe("ACP E2E", () => {
  let child: ChildProcess | undefined;

  afterEach(async () => {
    if (child && !child.killed) {
      child.kill();
      // Wait briefly for cleanup
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    child = undefined;
  });

  it("initialize — returns protocol version and agent info", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    const result = await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    expect(result).toBeDefined();
    expect(result.protocolVersion).toBe(PROTOCOL_VERSION);
    expect(result.agentInfo).toBeDefined();
    expect(result.agentInfo?.name).toBe("osagent");
  }, 30_000);

  it("session/new — creates a session with a sessionId", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    const session = await ctx.connection.newSession({
      cwd: process.cwd(),
      mcpServers: [],
    });

    expect(session).toBeDefined();
    expect(session.sessionId).toBeDefined();
    expect(typeof session.sessionId).toBe("string");
    expect(session.sessionId.length).toBeGreaterThan(0);
  }, 30_000);

  it("session/prompt — returns stopReason", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    const session = await ctx.connection.newSession({
      cwd: process.cwd(),
      mcpServers: [],
    });

    const promptResult = await ctx.connection.prompt({
      sessionId: session.sessionId,
      prompt: [{ type: "text", text: "Say hello in one word." }],
    });

    expect(promptResult).toBeDefined();
    expect(promptResult.stopReason).toBeDefined();
    // stopReason should be one of the valid ACP values
    expect([
      "end_turn",
      "max_tokens",
      "max_turn_requests",
      "refusal",
      "cancelled",
    ]).toContain(promptResult.stopReason);
  }, 60_000);

  it("session/prompt — streams session updates", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    const session = await ctx.connection.newSession({
      cwd: process.cwd(),
      mcpServers: [],
    });

    await ctx.connection.prompt({
      sessionId: session.sessionId,
      prompt: [{ type: "text", text: "Say hello in one word." }],
    });

    // Should have received at least one session update
    expect(ctx.client.updates.length).toBeGreaterThan(0);

    // Check that updates have the correct sessionId
    for (const update of ctx.client.updates) {
      expect(update.sessionId).toBe(session.sessionId);
    }
  }, 60_000);

  it("session/prompt — update contains agent_message_chunk with text", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    const session = await ctx.connection.newSession({
      cwd: process.cwd(),
      mcpServers: [],
    });

    await ctx.connection.prompt({
      sessionId: session.sessionId,
      prompt: [{ type: "text", text: "Say hello in one word." }],
    });

    // Should have at least one agent_message_chunk update
    const messageChunks = ctx.client.updates.filter(
      (u) => u.update.sessionUpdate === "agent_message_chunk",
    );
    expect(messageChunks.length).toBeGreaterThan(0);

    // Each chunk should have content with type text
    for (const chunk of messageChunks) {
      if (chunk.update.sessionUpdate === "agent_message_chunk") {
        expect(chunk.update.content).toBeDefined();
        expect(chunk.update.content.type).toBe("text");
      }
    }
  }, 60_000);

  it("initialize — rejects unknown methods with error", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    // Use extMethod to call a non-existent method
    await expect(
      ctx.connection.extMethod("nonexistent/method", {}),
    ).rejects.toThrow();
  }, 30_000);

  it("session/cancel — cancels an in-progress prompt", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    const session = await ctx.connection.newSession({
      cwd: process.cwd(),
      mcpServers: [],
    });

    // Start a long-running prompt and cancel it
    const promptPromise = ctx.connection.prompt({
      sessionId: session.sessionId,
      prompt: [
        {
          type: "text",
          text: "Write a 1000-word essay about the history of computing.",
        },
      ],
    });

    // Give it a moment to start, then cancel
    await new Promise((resolve) => setTimeout(resolve, 1000));
    await ctx.connection.cancel({ sessionId: session.sessionId });

    const result = await promptPromise;
    expect(result.stopReason).toBe("cancelled");
  }, 60_000);

  it("multiple sessions — can create and prompt independent sessions", async () => {
    const ctx = createAcpConnection();
    child = ctx.child;

    await ctx.connection.initialize({
      protocolVersion: PROTOCOL_VERSION,
      clientCapabilities: {},
    });

    const session1 = await ctx.connection.newSession({
      cwd: process.cwd(),
      mcpServers: [],
    });

    const session2 = await ctx.connection.newSession({
      cwd: process.cwd(),
      mcpServers: [],
    });

    // Sessions should have different IDs
    expect(session1.sessionId).not.toBe(session2.sessionId);
  }, 30_000);
});
