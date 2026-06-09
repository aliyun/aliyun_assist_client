import type { Notification, Request, ClientResponse } from "../messages.js";

/**
 * The agentic loop as a bidirectional async generator.
 * Yields: server messages (notifications and requests)
 * Receives: client responses to requests (or undefined for notifications)
 */
export type AgentLoop = AsyncGenerator<
  Notification | Request,
  void,
  ClientResponse | undefined
>;
