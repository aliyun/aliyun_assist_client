import { APIError, APIConnectionError, APIConnectionTimeoutError } from "openai";
import type { TurnError } from "protocol";

// ---------------------------------------------------------------------------
// TurnFailedError — structured error carrying a protocol TurnError
// ---------------------------------------------------------------------------

/**
 * Error subclass wrapping a protocol-level {@link TurnError}.
 *
 * Thrown by `AppServer.turnStart()` when a turn fails so that callers can
 * extract structured error information (`code`, `message`) instead of
 * relying on the raw `Error.message` string.
 */
export class TurnFailedError extends Error {
  readonly turnError: TurnError;

  constructor(turnError: TurnError) {
    super(turnError.message);
    this.name = "TurnFailedError";
    this.turnError = turnError;
  }
}

// ---------------------------------------------------------------------------
// classifyApiError — maps any error to a structured TurnError
// ---------------------------------------------------------------------------

/**
 * Classify an error (especially OpenAI SDK {@link APIError}) into a
 * protocol-conformant {@link TurnError}.
 *
 * Produces:
 * - A `code` string (e.g. "unauthorized", "server_overloaded")
 * - A human-readable `message` (e.g. "LLM provider returned HTTP 502 (Bad Gateway)")
 */
export function classifyApiError(error: unknown): TurnError {
  if (error instanceof APIError) {
    const code = mapStatusToCode(error);
    const message = buildMessage(error);
    return { code, message };
  }

  if (error instanceof Error) {
    return { code: "other", message: error.message };
  }

  return { code: "other", message: String(error) };
}

// ---------------------------------------------------------------------------
// formatTurnError — format any error for CLI display
// ---------------------------------------------------------------------------

/**
 * Format an error for user-facing CLI output.
 *
 * When the error is a {@link TurnFailedError}, formats the structured
 * `TurnError` message. Otherwise falls back to `Error.message` or
 * `String(error)`.
 *
 * Returns an array of lines (without trailing newlines) for the caller
 * to join or print as needed.
 */
export function formatTurnError(error: unknown): string[] {
  if (error instanceof TurnFailedError) {
    return [`[Error] ${error.turnError.message}`];
  }

  const message = error instanceof Error ? error.message : String(error);
  return [`[Error] ${message}`];
}

// ---------------------------------------------------------------------------
// Internal: APIError → code string mapping
// ---------------------------------------------------------------------------

function mapStatusToCode(error: APIError): string {
  if (error instanceof APIConnectionTimeoutError) return "connection_timeout";
  if (error instanceof APIConnectionError) return "http_connection_failed";

  const status = error.status;
  if (status === undefined || status === null) return "other";

  // Check for context window exceeded via error body
  const errorBody = error.error as { code?: string; type?: string } | undefined;
  if (
    errorBody?.code === "context_length_exceeded" ||
    errorBody?.type === "context_length_exceeded"
  ) {
    return "context_window_exceeded";
  }

  switch (status) {
    case 400: return "bad_request";
    case 401: return "unauthorized";
    case 403: return "unauthorized";       // Permission denied → closest match
    case 429: return "usage_limit_exceeded";
    case 500: return "internal_server_error";
    case 503: return "server_overloaded";
    default:
      if (status >= 400) return "http_connection_failed";
      return "other";
  }
}

// ---------------------------------------------------------------------------
// Internal: human-readable message from APIError structured fields
// ---------------------------------------------------------------------------

function buildMessage(error: APIError): string {
  const parts: string[] = [];

  if (error instanceof APIConnectionTimeoutError) {
    parts.push("Connection to LLM provider timed out");
  } else if (error instanceof APIConnectionError) {
    parts.push("Failed to connect to LLM provider");
  } else if (error.status !== undefined) {
    const text = httpStatusText(error.status);
    parts.push(`LLM provider returned HTTP ${error.status}${text ? ` (${text})` : ""}`);
  } else {
    parts.push("LLM provider request failed");
  }

  // Append provider error message from JSON body (not raw HTML)
  const errorBody = error.error as { message?: string; code?: string; type?: string } | undefined;
  if (errorBody?.message) {
    parts.push(errorBody.message);
  } else if (errorBody?.code) {
    parts.push(`Code: ${errorBody.code}`);
  }

  return parts.join(" — ");
}

// ---------------------------------------------------------------------------
// Internal: HTTP status → human text
// ---------------------------------------------------------------------------

function httpStatusText(status: number): string | null {
  const map: Record<number, string> = {
    400: "Bad Request",
    401: "Unauthorized",
    403: "Forbidden",
    404: "Not Found",
    409: "Conflict",
    422: "Unprocessable Entity",
    429: "Rate Limited",
    500: "Internal Server Error",
    502: "Bad Gateway",
    503: "Service Unavailable",
    504: "Gateway Timeout",
  };
  return map[status] ?? null;
}
