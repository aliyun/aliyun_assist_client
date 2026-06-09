import type { AppServer } from "core";
import type {
  ClientResponse,
  CommandExecutionApprovalResponse,
  Item,
  Notification,
  Request,
} from "protocol";
import { useReducer, useCallback, useRef } from "react";

// ---------------------------------------------------------------------------
// Approval decision (moved from Approval.tsx — shared between hook and UI)
// ---------------------------------------------------------------------------

export type ApprovalDecision = "approved" | "denied" | "always_approve";

// ---------------------------------------------------------------------------
// State types
// ---------------------------------------------------------------------------

export type TurnPhase =
  | "idle"
  | "thinking"
  | "streaming"
  | "awaiting_approval"
  | "tool_running";

export type ToolCallStatus =
  | "pending_approval"
  | "running"
  | "completed"
  | "failed"
  | "denied";

export type ToolCallState = {
  id: string;
  toolName: string;
  itemType: "commandExecution" | "toolCall" | "mcpToolCall";
  status: ToolCallStatus;
  output?: string;
  /** Raw JSON arguments string (present for toolCall / mcpToolCall) */
  arguments?: string;
  // Command execution specific (present when itemType === "commandExecution")
  toolType?: "local" | "remote";
  command?: string;
  target?: string;
  exitCode?: number | null;
  durationMs?: number | null;
  autoApproved?: boolean;
};

/**
 * A single segment of a turn's conversation flow. Segments are stored in
 * the order they were received from the server so that text and tool-call
 * blocks render inline at the point they occurred.
 */
export type TurnSegment =
  | { type: "text"; content: string }
  | { type: "toolCall"; toolCall: ToolCallState };

/**
 * An append-only block rendered inside Ink's `<Static>` area. Each entry is
 * emitted to the terminal exactly once and never re-rendered, so anything we
 * want stable across streaming deltas (notably the user prompt and any
 * already-finalized segment) belongs here.
 *
 * Segments are frozen progressively as soon as they are finalized, so a
 * completed tool-call box stops being part of Ink's live render area mid-turn
 * and no longer ghosts on every streaming delta.
 */
export type StaticItem =
  | { kind: "prompt"; key: string; userInput: string }
  | { kind: "segment"; key: string; segment: TurnSegment }
  | { kind: "turnEnd"; key: string };

export type ApprovalState = {
  requestId: string;
  itemId: string;
  command: string;
  cwd?: string;
};

export type TurnState = {
  phase: TurnPhase;
  userInput: string;
  segments: TurnSegment[];
  /**
   * Cursor into `segments`: entries with index < `frozenSegmentCount` have
   * already been pushed to `staticItems` and must NOT be rendered in the
   * live area. Only segments at index >= `frozenSegmentCount` are still
   * "active" and may be redrawn on subsequent state updates.
   */
  frozenSegmentCount: number;
  approval: ApprovalState | null;
  /**
   * Append-only list of frozen blocks. Grows progressively: the user prompt
   * is frozen on turn start, each segment is frozen as soon as it is
   * finalized (a later segment starts, or a tool-call reaches a terminal
   * status), and a `turnEnd` spacer is appended on turn completion. The
   * reference identity is stable for items that have already been emitted,
   * which keeps `<Static>` from re-emitting them.
   */
  staticItems: StaticItem[];
  /** Monotonic counter used to mint stable keys for `staticItems`. */
  nextStaticKey: number;
  error: string | null;
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Parse the display command from the server to determine tool type and extract
 * the actual command string. Remote commands are prefixed with
 * `[Remote: instance_id@region_id]`.
 */
function parseCommand(rawCommand: string): {
  toolType: "local" | "remote";
  command: string;
  target?: string;
} {
  const remoteMatch = rawCommand.match(/^\[Remote:\s*(.+?)\]\s*(.+)$/s);
  if (remoteMatch) {
    return { toolType: "remote", command: remoteMatch[2]!, target: remoteMatch[1]! };
  }
  return { toolType: "local", command: rawCommand };
}

/**
 * Append a text delta to the segments list. If the last segment is a text
 * block, extend it; otherwise push a new text segment so that text following
 * a tool call starts on its own block.
 */
function appendTextDelta(segments: TurnSegment[], delta: string): TurnSegment[] {
  if (segments.length > 0) {
    const last = segments[segments.length - 1]!;
    if (last.type === "text") {
      return [
        ...segments.slice(0, -1),
        { type: "text", content: last.content + delta },
      ];
    }
  }
  return [...segments, { type: "text", content: delta }];
}

/**
 * Update the toolCall segment whose id matches `id` using the provided
 * mapping function. Non-matching segments are returned unchanged.
 */
function updateToolCall(
  segments: TurnSegment[],
  id: string,
  update: (tc: ToolCallState) => ToolCallState,
): TurnSegment[] {
  return segments.map((seg) => {
    if (seg.type !== "toolCall" || seg.toolCall.id !== id) return seg;
    return { type: "toolCall", toolCall: update(seg.toolCall) };
  });
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

type TurnAction =
  | { type: "TURN_STARTED"; userInput: string }
  | { type: "STREAM_DELTA"; delta: string }
  | { type: "ITEM_STARTED"; item: Item }
  | { type: "ITEM_COMPLETED"; item: Item }
  | { type: "APPROVAL_REQUESTED"; approval: ApprovalState }
  | { type: "APPROVAL_RESOLVED"; decision: ApprovalDecision; itemId: string }
  | { type: "APPROVAL_AUTO"; itemId: string }
  | { type: "TURN_COMPLETED" }
  | { type: "TURN_ERROR"; error: string };

// ---------------------------------------------------------------------------
// Reducer
// ---------------------------------------------------------------------------

const INITIAL_STATE: TurnState = {
  phase: "idle",
  userInput: "",
  segments: [],
  frozenSegmentCount: 0,
  approval: null,
  staticItems: [],
  nextStaticKey: 0,
  error: null,
};

/**
 * Returns true if a tool call has reached a terminal state and will not
 * mutate any further. Pending-approval and running calls are still active.
 */
function isToolCallTerminal(status: ToolCallStatus): boolean {
  return status === "completed" || status === "failed" || status === "denied";
}

/**
 * Walk the segments array starting at `frozenSegmentCount` and freeze every
 * leading segment that is no longer subject to change:
 *   - any non-trailing segment is by definition complete (a later segment
 *     has been pushed after it),
 *   - a trailing tool-call segment is complete once its status is terminal.
 *
 * A trailing text segment is never frozen here because more deltas may still
 * arrive; it is frozen either when the next segment starts (handled by this
 * same walk on the next reduction) or when the turn ends (`freezeAllRemaining`).
 */
function progressivelyFreeze(state: TurnState): TurnState {
  let frozenCount = state.frozenSegmentCount;
  const additions: StaticItem[] = [];
  let nextKey = state.nextStaticKey;

  while (frozenCount < state.segments.length) {
    const seg = state.segments[frozenCount]!;
    const isTrailing = frozenCount === state.segments.length - 1;
    let canFreeze = !isTrailing;
    if (isTrailing && seg.type === "toolCall" && isToolCallTerminal(seg.toolCall.status)) {
      canFreeze = true;
    }
    if (!canFreeze) break;
    additions.push({ kind: "segment", key: `segment-${nextKey}`, segment: seg });
    nextKey += 1;
    frozenCount += 1;
  }

  if (additions.length === 0) return state;
  return {
    ...state,
    staticItems: [...state.staticItems, ...additions],
    nextStaticKey: nextKey,
    frozenSegmentCount: frozenCount,
  };
}

/**
 * Freeze every remaining unfrozen segment unconditionally. Used at turn end
 * (success or error) to flush whatever is left in the live area into the
 * static scrollback before resetting state.
 */
function freezeAllRemaining(state: TurnState): TurnState {
  if (state.frozenSegmentCount >= state.segments.length) return state;
  const additions: StaticItem[] = [];
  let nextKey = state.nextStaticKey;
  for (let i = state.frozenSegmentCount; i < state.segments.length; i += 1) {
    additions.push({ kind: "segment", key: `segment-${nextKey}`, segment: state.segments[i]! });
    nextKey += 1;
  }
  return {
    ...state,
    staticItems: [...state.staticItems, ...additions],
    nextStaticKey: nextKey,
    frozenSegmentCount: state.segments.length,
  };
}

function turnReducer(state: TurnState, action: TurnAction): TurnState {
  const next = reduceCore(state, action);
  // Cases that explicitly handle freezing themselves (TURN_COMPLETED /
  // TURN_ERROR call freezeAllRemaining) reset the segments cursor and are
  // safe no-ops here. Every other action runs through progressivelyFreeze so
  // that a segment is pushed to <Static> the moment it stops being mutable.
  return progressivelyFreeze(next);
}

function reduceCore(state: TurnState, action: TurnAction): TurnState {
  switch (action.type) {
    case "TURN_STARTED": {
      // Freeze the user prompt into <Static> immediately so it is emitted
      // once and never re-rendered as streaming deltas arrive. Rendering it
      // in the live area instead would cause Ink to clear+redraw it on every
      // state update, which xterm.js (VS Code / Qoder) renders as ghost
      // duplicate lines.
      const promptItem: StaticItem = {
        kind: "prompt",
        key: `prompt-${state.nextStaticKey}`,
        userInput: action.userInput,
      };
      return {
        ...state,
        phase: "thinking",
        userInput: action.userInput,
        segments: [],
        frozenSegmentCount: 0,
        approval: null,
        staticItems: [...state.staticItems, promptItem],
        nextStaticKey: state.nextStaticKey + 1,
        error: null,
      };
    }

    case "STREAM_DELTA":
      return {
        ...state,
        phase: "streaming",
        segments: appendTextDelta(state.segments, action.delta),
      };

    case "ITEM_STARTED": {
      const item = action.item;
      if (item.type === "commandExecution") {
        const parsed = parseCommand(item.command);
        const toolCall: ToolCallState = {
          id: item.id,
          toolName: item.command,
          itemType: "commandExecution",
          status: "running",
          toolType: parsed.toolType,
          command: parsed.command,
          target: parsed.target,
          exitCode: null,
          durationMs: null,
          autoApproved: false,
        };
        return {
          ...state,
          phase: "tool_running",
          segments: [...state.segments, { type: "toolCall", toolCall }],
        };
      }
      if (item.type === "toolCall" || item.type === "mcpToolCall") {
        const toolName = item.type === "mcpToolCall"
          ? `${item.serverName}/${item.toolName}`
          : item.toolName;
        const toolCall: ToolCallState = {
          id: item.id,
          toolName,
          itemType: item.type,
          status: "running",
          arguments: item.arguments || undefined,
        };
        return {
          ...state,
          phase: "tool_running",
          segments: [...state.segments, { type: "toolCall", toolCall }],
        };
      }
      if (item.type === "reasoning") {
        return { ...state, phase: "thinking" };
      }
      return state;
    }

    case "ITEM_COMPLETED": {
      const item = action.item;
      if (
        item.type === "commandExecution" ||
        item.type === "toolCall" ||
        item.type === "mcpToolCall"
      ) {
        return {
          ...state,
          segments: updateToolCall(state.segments, item.id, (tc) => {
            // Don't overwrite "denied" status from approval resolution
            if (tc.status === "denied") return tc;
            const newStatus: ToolCallStatus = item.status === "completed" ? "completed" : "failed";
            if (item.type === "commandExecution") {
              return {
                ...tc,
                status: newStatus,
                output: item.output,
                exitCode: item.exitCode,
                durationMs: item.durationMs,
              };
            }
            return { ...tc, status: newStatus, output: item.output };
          }),
        };
      }
      return state;
    }

    case "APPROVAL_REQUESTED":
      return {
        ...state,
        phase: "awaiting_approval",
        approval: action.approval,
        segments: updateToolCall(state.segments, action.approval.itemId, (tc) => ({
          ...tc,
          status: "pending_approval",
        })),
      };

    case "APPROVAL_RESOLVED": {
      const newStatus: ToolCallStatus = action.decision === "denied" ? "denied" : "running";
      return {
        ...state,
        phase: "tool_running",
        approval: null,
        segments: updateToolCall(state.segments, action.itemId, (tc) => ({
          ...tc,
          status: newStatus,
        })),
      };
    }

    case "APPROVAL_AUTO":
      return {
        ...state,
        segments: updateToolCall(state.segments, action.itemId, (tc) => ({
          ...tc,
          autoApproved: true,
        })),
      };

    case "TURN_COMPLETED": {
      // Flush any segments still in the live area (typically a trailing text
      // segment) into <Static> and append a one-line spacer to visually
      // separate this turn from the next prompt.
      const flushed = freezeAllRemaining(state);
      const turnEndItem: StaticItem = {
        kind: "turnEnd",
        key: `turn-end-${flushed.nextStaticKey}`,
      };
      return {
        ...flushed,
        phase: "idle",
        staticItems: [...flushed.staticItems, turnEndItem],
        nextStaticKey: flushed.nextStaticKey + 1,
        userInput: "",
        segments: [],
        frozenSegmentCount: 0,
        approval: null,
      };
    }

    case "TURN_ERROR": {
      const flushed = freezeAllRemaining(state);
      const turnEndItem: StaticItem = {
        kind: "turnEnd",
        key: `turn-end-${flushed.nextStaticKey}`,
      };
      return {
        ...flushed,
        phase: "idle",
        error: action.error,
        staticItems: [...flushed.staticItems, turnEndItem],
        nextStaticKey: flushed.nextStaticKey + 1,
        userInput: "",
        segments: [],
        frozenSegmentCount: 0,
        approval: null,
      };
    }

    default:
      return state;
  }
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export type UseTurnResult = {
  state: TurnState;
  startTurn: (input: string) => void;
  resolveApproval: (decision: ApprovalDecision) => void;
};

export function useTurn(
  server: AppServer,
  threadId: string,
  approvalMode: "prompt" | "yolo",
): UseTurnResult {
  const [state, dispatch] = useReducer(turnReducer, INITIAL_STATE);

  // Ref to hold the resolve function for the current approval request
  const approvalResolveRef = useRef<((response: ClientResponse) => void) | null>(null);

  const startTurn = useCallback(
    (input: string) => {
      dispatch({ type: "TURN_STARTED", userInput: input });

      // Run the turn in the background (not awaited — updates are dispatched)
      void (async () => {
        try {
          const gen = server.turnStart({ threadId, input: [{ type: "text", text: input }] });
          let result = await gen.next();

          while (!result.done) {
            const msg = result.value;

            if ("id" in msg) {
              // Request — handle approval
              const request = msg as Request;
              const approvalParams = request.params;
              const itemId = (approvalParams as { itemId?: string }).itemId ?? "";

              if (approvalMode === "yolo") {
                // Auto-approve in YOLO mode
                dispatch({ type: "APPROVAL_AUTO", itemId });
                const clientResponse: ClientResponse = {
                  id: request.id,
                  result: { decision: "approved" } satisfies CommandExecutionApprovalResponse,
                };
                result = await gen.next(clientResponse);
              } else {
                // Prompt user — pause the generator until user decides
                dispatch({
                  type: "APPROVAL_REQUESTED",
                  approval: {
                    requestId: request.id,
                    itemId,
                    command: (approvalParams as { command?: string }).command ?? "(unknown)",
                    cwd: (approvalParams as { cwd?: string }).cwd,
                  },
                });

                // Wait for user decision
                const clientResponse = await new Promise<ClientResponse>((resolve) => {
                  approvalResolveRef.current = resolve;
                });

                const resolvedDecision =
                  (clientResponse.result as CommandExecutionApprovalResponse).decision;
                dispatch({ type: "APPROVAL_RESOLVED", decision: resolvedDecision, itemId });
                result = await gen.next(clientResponse);
              }
            } else {
              // Notification
              const notification = msg as Notification;
              handleNotification(notification, dispatch);
              result = await gen.next(undefined);
            }
          }

          dispatch({ type: "TURN_COMPLETED" });
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          dispatch({ type: "TURN_ERROR", error: message });
        }
      })();
    },
    [server, threadId, approvalMode],
  );

  const resolveApproval = useCallback(
    (decision: ApprovalDecision) => {
      if (!state.approval || !approvalResolveRef.current) return;

      const clientResponse: ClientResponse = {
        id: state.approval.requestId,
        result: { decision } satisfies CommandExecutionApprovalResponse,
      };
      approvalResolveRef.current(clientResponse);
      approvalResolveRef.current = null;
    },
    [state.approval],
  );

  return { state, startTurn, resolveApproval };
}

// ---------------------------------------------------------------------------
// Notification handler (dispatches actions to the reducer)
// ---------------------------------------------------------------------------

function handleNotification(n: Notification, dispatch: React.Dispatch<TurnAction>): void {
  switch (n.method) {
    case "item/delta":
      if (n.params.delta.type === "agentMessage") {
        dispatch({ type: "STREAM_DELTA", delta: n.params.delta.delta });
      }
      break;
    case "item/started":
      dispatch({ type: "ITEM_STARTED", item: n.params.item });
      break;
    case "item/completed":
      dispatch({ type: "ITEM_COMPLETED", item: n.params.item });
      break;
    case "turn/started":
    case "turn/completed":
    case "request/resolved":
    case "error":
      break;
    default:
      break;
  }
}
