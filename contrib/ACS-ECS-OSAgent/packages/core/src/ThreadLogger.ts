// ThreadLogger — append-only JSONL persistence for the agentic loop.
//
// Each non-ephemeral Thread gets its own log file at:
//
//     <threadsDir>/<uuidv7>.jsonl
//
// where `<uuidv7>` is the bare UUIDv7 portion of the threadId (i.e. with the
// `thread_` prefix stripped). One JSON object per line; writes are
// synchronous so a crash loses at most the last unflushed record.
//
// Record envelope:
//
//     { "v": 1, "ts": "<ISO-8601 ms>", "threadId": "thread_<uuid>",
//       "type": "<discriminator>", "data": { ... } }
//
// See docs/feature-specs/2026-05-08-session-persistence.md for the full
// catalogue of record types and their `data` shapes.

import { appendFileSync, closeSync, mkdirSync, openSync } from "node:fs";
import { join } from "node:path";

export type ThreadRecordType =
  | "thread_started"
  | "turn_started"
  | "llm_request"
  | "llm_response"
  | "tool_started"
  | "approval_requested"
  | "approval_resolved"
  | "tool_completed"
  | "turn_completed"
  | "turn_failed";

export type ThreadRecord = {
  v: 1;
  ts: string;
  threadId: string;
  type: ThreadRecordType;
  data: unknown;
};

export class ThreadLogger {
  public readonly path: string;
  private readonly threadId: string;
  private fd: number | null;

  constructor(threadId: string, threadsDir: string) {
    this.threadId = threadId;
    mkdirSync(threadsDir, { recursive: true });

    // Filename strips the `thread_` prefix — the directory already
    // establishes the entity type, so repeating it would be redundant.
    const filename = threadId.startsWith("thread_")
      ? threadId.slice("thread_".length)
      : threadId;
    this.path = join(threadsDir, `${filename}.jsonl`);
    this.fd = openSync(this.path, "a");
  }

  append(type: ThreadRecordType, data: unknown): void {
    if (this.fd === null) return;
    const record: ThreadRecord = {
      v: 1,
      ts: new Date().toISOString(),
      threadId: this.threadId,
      type,
      data,
    };
    appendFileSync(this.fd, JSON.stringify(record) + "\n");
  }

  close(): void {
    if (this.fd === null) return;
    closeSync(this.fd);
    this.fd = null;
  }
}
