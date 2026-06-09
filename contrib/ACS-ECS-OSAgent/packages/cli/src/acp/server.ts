import { Readable, Writable } from "node:stream";

import {
  AgentSideConnection,
  ndJsonStream,
  PROTOCOL_VERSION,
} from "@agentclientprotocol/sdk";
import type {
  Agent,
  InitializeRequest,
  InitializeResponse,
  NewSessionRequest,
  NewSessionResponse,
  PromptRequest,
  PromptResponse,
  CancelNotification,
  AuthenticateRequest,
  AuthenticateResponse,
  StopReason,
  SessionNotification,
  RequestPermissionResponse,
} from "@agentclientprotocol/sdk";
import type { AppServer } from "core";
import type { ClientResponse } from "protocol";

import { toSessionUpdate } from "./mapping.js";

/**
 * Start the ACP stdio server using the official SDK.
 * The AgentSideConnection handles all JSON-RPC dispatch and type conformance.
 */
export async function startAcpServer(appServer: AppServer): Promise<void> {
  // Disable terminal line buffering, echo, and signal handling when stdin is a
  // TTY so JSON-RPC frames are delivered to the SDK byte-for-byte. When stdin
  // is a pipe (the common case), setRawMode is unavailable and must be skipped.
  if (process.stdin.isTTY) {
    process.stdin.setRawMode(true);
  }

  // Convert Node.js stdio → Web Streams for the SDK
  const output = Writable.toWeb(process.stdout) as WritableStream<Uint8Array>;
  const input = Readable.toWeb(process.stdin) as ReadableStream<Uint8Array>;
  const stream = ndJsonStream(output, input);

  /** Cancellation flags keyed by sessionId */
  const cancelFlags = new Map<string, { cancelled: boolean }>();

  const connection = new AgentSideConnection((conn): Agent => {
    return {
      async initialize(_params: InitializeRequest): Promise<InitializeResponse> {
        return {
          protocolVersion: PROTOCOL_VERSION,
          agentCapabilities: {},
          agentInfo: { name: "osagent", version: "0.0.1" },
        };
      },

      async newSession(params: NewSessionRequest): Promise<NewSessionResponse> {
        const { threadId } = await appServer.threadStart({
          cwd: params.cwd,
        });
        return { sessionId: threadId };
      },

      async prompt(params: PromptRequest): Promise<PromptResponse> {
        const { sessionId, prompt } = params;
        const cancelFlag = { cancelled: false };
        cancelFlags.set(sessionId, cancelFlag);

        // Extract text from prompt ContentBlock[]
        const input = prompt
          .filter((block) => block.type === "text")
          .map((block) => ({
            type: "text" as const,
            text: (block as { type: "text"; text: string }).text,
          }));

        const loop = appServer.turnStart({ threadId: sessionId, input });
        let stopReason: StopReason = "end_turn";

        try {
          let result = await loop.next();
          while (!result.done) {
            if (cancelFlag.cancelled) {
              await loop.return(undefined);
              stopReason = "cancelled";
              break;
            }

            const msg = result.value;

            if ("id" in msg) {
              // It's a Request — request permission from client via SDK
              const permResponse: RequestPermissionResponse = await conn.requestPermission({
                sessionId,
                toolCall: {
                  toolCallId: String(msg.id),
                  title: `Execute: ${(msg.params as { command?: string }).command ?? "command"}`,
                  status: "in_progress",
                  kind: "execute",
                },
                options: [
                  { optionId: "allow", name: "Allow", kind: "allow_once" },
                  { optionId: "deny", name: "Deny", kind: "reject_once" },
                ],
              });

              const decision =
                permResponse.outcome.outcome === "selected" ? "approved" as const : "denied" as const;
              const clientResponse: ClientResponse = {
                id: String(msg.id),
                result: { decision },
              };
              result = await loop.next(clientResponse);
            } else {
              // It's a Notification — map to ACP session update and send
              const update = toSessionUpdate(msg);
              if (update) {
                const notification: SessionNotification = {
                  sessionId,
                  update,
                };
                await conn.sessionUpdate(notification);
              }

              // Capture stop reason from turn/completed
              if (msg.method === "turn/completed") {
                const sr = (msg.params as { stopReason?: string }).stopReason;
                if (sr === "end_turn" || sr === "cancelled") {
                  stopReason = sr;
                }
              }

              result = await loop.next(undefined);
            }
          }
        } catch {
          stopReason = "end_turn";
        } finally {
          cancelFlags.delete(sessionId);
        }

        return { stopReason };
      },

      async cancel(params: CancelNotification): Promise<void> {
        const flag = cancelFlags.get(params.sessionId);
        if (flag) {
          flag.cancelled = true;
        }
      },

      async authenticate(_params: AuthenticateRequest): Promise<AuthenticateResponse> {
        return {};
      },
    };
  }, stream);

  // Wait for the connection to close (stdin EOF)
  await connection.closed;
}
