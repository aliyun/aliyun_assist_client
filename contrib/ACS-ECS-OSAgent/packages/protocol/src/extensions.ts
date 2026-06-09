import type { Item } from "./items.js";
import type { TurnStatus, TurnError } from "./lifecycle.js";

/** Thread forking (planned). */
export type ThreadForkParams = {
  sourceThreadId: string;
  fromTurnId?: string;
};

/** Skill activation. */
export type SkillActivateParams = {
  skillName: string;
  arguments: Record<string, unknown>;
};

export type SkillActivateResult = {
  content: string;
  mcpServers?: Record<string, unknown>;
};

/** Thread persistence events. */
export type ThreadLogEvent = {
  threadId: string;
  event: Item | TurnLifecycleEvent;
};

export type TurnLifecycleEvent =
  | { type: "turn_started"; turnId: string }
  | { type: "turn_completed"; turnId: string; status: TurnStatus }
  | { type: "turn_failed"; turnId: string; error: TurnError };
