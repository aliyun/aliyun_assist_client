import type { Item } from "./items.js";

export type Turn = {
  id: string;
  status: TurnStatus;
  items: Item[];
  error: TurnError | null;
};

export type TurnStatus = "in_progress" | "completed" | "failed" | "cancelled";

export type StopReason = "end_turn" | "max_tokens" | "cancelled" | "error";

export type TurnError = {
  code: string;
  message: string;
};
