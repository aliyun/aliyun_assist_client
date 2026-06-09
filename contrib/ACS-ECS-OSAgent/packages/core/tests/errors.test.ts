import { APIError, APIConnectionError, APIConnectionTimeoutError } from "openai";
import { describe, it, expect } from "vitest";

import {
  TurnFailedError,
  classifyApiError,
  formatTurnError,
} from "../src/errors.js";

// ── TurnFailedError ─────────────────────────────────────────────────

describe("TurnFailedError", () => {
  it("wraps a TurnError with correct name and message", () => {
    const turnError = { code: "other", message: "Something went wrong" };
    const err = new TurnFailedError(turnError);
    expect(err).toBeInstanceOf(Error);
    expect(err.name).toBe("TurnFailedError");
    expect(err.message).toBe("Something went wrong");
    expect(err.turnError).toBe(turnError);
  });

  it("preserves code and message on turnError", () => {
    const turnError = { code: "bad_request", message: "Failure" };
    const err = new TurnFailedError(turnError);
    expect(err.turnError.code).toBe("bad_request");
    expect(err.turnError.message).toBe("Failure");
  });
});

// ── classifyApiError ────────────────────────────────────────────────

describe("classifyApiError", () => {
  it("classifies a plain Error as 'other'", () => {
    const result = classifyApiError(new Error("generic error"));
    expect(result.code).toBe("other");
    expect(result.message).toBe("generic error");
  });

  it("classifies a string as 'other'", () => {
    const result = classifyApiError("string error");
    expect(result.code).toBe("other");
    expect(result.message).toBe("string error");
  });

  it("classifies APIConnectionTimeoutError", () => {
    const err = new APIConnectionTimeoutError({ message: "timeout" });
    const result = classifyApiError(err);
    expect(result.code).toBe("connection_timeout");
    expect(result.message).toContain("timed out");
  });

  it("classifies APIConnectionError", () => {
    const err = new APIConnectionError({ message: "connection refused", cause: undefined });
    const result = classifyApiError(err);
    expect(result.code).toBe("http_connection_failed");
    expect(result.message).toContain("connect");
  });

  it("classifies 401 as unauthorized", () => {
    const err = new APIError(401, { message: "invalid key" }, "Unauthorized", {});
    const result = classifyApiError(err);
    expect(result.code).toBe("unauthorized");
  });

  it("classifies 403 as unauthorized", () => {
    const err = new APIError(403, { message: "forbidden" }, "Forbidden", {});
    const result = classifyApiError(err);
    expect(result.code).toBe("unauthorized");
  });

  it("classifies 429 as usage_limit_exceeded", () => {
    const err = new APIError(429, { message: "rate limit" }, "Too Many Requests", {});
    const result = classifyApiError(err);
    expect(result.code).toBe("usage_limit_exceeded");
  });

  it("classifies 400 as bad_request", () => {
    const err = new APIError(400, { message: "bad request" }, "Bad Request", {});
    const result = classifyApiError(err);
    expect(result.code).toBe("bad_request");
  });

  it("classifies 500 as internal_server_error", () => {
    const err = new APIError(500, { message: "server error" }, "Internal Server Error", {});
    const result = classifyApiError(err);
    expect(result.code).toBe("internal_server_error");
  });

  it("classifies 503 as server_overloaded", () => {
    const err = new APIError(503, { message: "overloaded" }, "Service Unavailable", {});
    const result = classifyApiError(err);
    expect(result.code).toBe("server_overloaded");
  });

  it("classifies context_length_exceeded from error body", () => {
    const err = new APIError(
      400,
      { code: "context_length_exceeded", message: "too long" },
      "Bad Request",
      {},
    );
    const result = classifyApiError(err);
    expect(result.code).toBe("context_window_exceeded");
  });

  it("classifies unknown 4xx as http_connection_failed", () => {
    const err = new APIError(418, { message: "teapot" }, "I'm a teapot", {});
    const result = classifyApiError(err);
    expect(result.code).toBe("http_connection_failed");
  });
});

// ── formatTurnError ─────────────────────────────────────────────────

describe("formatTurnError", () => {
  it("formats TurnFailedError message", () => {
    const err = new TurnFailedError({
      code: "other",
      message: "HTTP 502",
    });
    const lines = formatTurnError(err);
    expect(lines).toHaveLength(1);
    expect(lines[0]).toContain("[Error]");
    expect(lines[0]).toContain("HTTP 502");
  });

  it("formats TurnFailedError without extra lines", () => {
    const err = new TurnFailedError({
      code: "other",
      message: "Something failed",
    });
    const lines = formatTurnError(err);
    expect(lines).toHaveLength(1);
    expect(lines[0]).toContain("Something failed");
  });

  it("formats plain Error", () => {
    const lines = formatTurnError(new Error("plain error"));
    expect(lines).toHaveLength(1);
    expect(lines[0]).toContain("plain error");
  });

  it("formats non-Error value", () => {
    const lines = formatTurnError("string error");
    expect(lines).toHaveLength(1);
    expect(lines[0]).toContain("string error");
  });
});
