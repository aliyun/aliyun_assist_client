import type { CommandExecutionApprovalParams, CommandExecutionApprovalResponse } from "./approval.js";
import type { Item } from "./items.js";
import type { TurnStatus, StopReason, TurnError } from "./lifecycle.js";
import type { ItemDelta } from "./streaming.js";

/** Core lifecycle notification methods. */
export type NotificationMethod =
  | "turn/started"
  | "turn/completed"
  | "item/started"
  | "item/completed"
  | "item/delta"
  | "request/resolved"
  | "error"
  | `_osagent/${string}`;

/** Server request methods requiring client response. */
export type RequestMethod =
  | "approval/command_execution"
  | `_osagent/${string}`;

/** Maps each notification method to its params type. */
export interface NotificationParamsMap {
  "turn/started": TurnStartedParams;
  "turn/completed": TurnCompletedParams;
  "item/started": ItemStartedParams;
  "item/completed": ItemCompletedParams;
  "item/delta": ItemDeltaParams;
  "request/resolved": RequestResolvedParams;
  "error": ErrorParams;
  [key: `_osagent/${string}`]: Record<string, unknown>;
}

/** Maps each request method to its params type. */
export interface RequestParamsMap {
  "approval/command_execution": CommandExecutionApprovalParams;
  [key: `_osagent/${string}`]: Record<string, unknown>;
}

/** Maps each request method to its response result type. */
export interface ResponseResultMap {
  "approval/command_execution": CommandExecutionApprovalResponse;
  [key: `_osagent/${string}`]: Record<string, unknown>;
}

/**
 * A server-initiated message that does not expect a response.
 * Discriminated by `method` field.
 */
export type Notification = {
  [M in NotificationMethod]: {
    method: M;
    params: NotificationParamsMap[M];
  };
}[NotificationMethod];

/**
 * A server-initiated message that requires a client response.
 * Distinguished from Notification by the presence of `id`.
 */
export type Request = {
  [M in RequestMethod]: {
    method: M;
    id: string;
    params: RequestParamsMap[M];
  };
}[RequestMethod];

/** Union of all server-to-client messages. */
export type ServerMessage = Notification | Request;

/** Client's response to a server Request. */
export type ClientResponse = {
  id: string;
  result: ResponseResultMap[RequestMethod];
};

// --- Params types ---

export type TurnStartedParams = {
  threadId: string;
  turnId: string;
};

export type TurnCompletedParams = {
  threadId: string;
  turnId: string;
  status: TurnStatus;
  stopReason: StopReason;
  error: TurnError | null;
};

export type ItemStartedParams = {
  threadId: string;
  turnId: string;
  item: Item;
};

export type ItemCompletedParams = {
  threadId: string;
  turnId: string;
  item: Item;
};

export type ItemDeltaParams = {
  threadId: string;
  turnId: string;
  itemId: string;
  delta: ItemDelta;
};

export type RequestResolvedParams = {
  requestId: string;
  resolution: "approved" | "denied";
};

export type ErrorParams = {
  code: string;
  message: string;
  data?: unknown;
};
