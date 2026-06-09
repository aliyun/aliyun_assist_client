import { describe, it, expect } from "vitest";

import { interpolateObject } from "../src/interpolation.js";

describe("interpolateObject", () => {
  // ── Primitives ──────────────────────────────────────────────────

  it("passes null through unchanged", () => {
    expect(interpolateObject(null, {}, "test")).toBeNull();
  });

  it("passes undefined through unchanged", () => {
    expect(interpolateObject(undefined, {}, "test")).toBeUndefined();
  });

  it("passes numbers through unchanged", () => {
    expect(interpolateObject(42, {}, "test")).toBe(42);
  });

  it("passes booleans through unchanged", () => {
    expect(interpolateObject(true, {}, "test")).toBe(true);
  });

  // ── Strings ─────────────────────────────────────────────────────

  it("returns plain string unchanged (no placeholders)", () => {
    expect(interpolateObject("hello world", {}, "test")).toBe("hello world");
  });

  it("interpolates a single variable", () => {
    const result = interpolateObject("Hello {{ name }}", { name: "World" }, "test");
    expect(result).toBe("Hello World");
  });

  it("interpolates multiple variables", () => {
    const result = interpolateObject(
      "{{ greeting }}, {{ name }}!",
      { greeting: "Hi", name: "Alice" },
      "test",
    );
    expect(result).toBe("Hi, Alice!");
  });

  it("throws on missing variable with helpful message", () => {
    expect(() => interpolateObject("{{ missing }}", {}, "my-skill")).toThrow(
      /my-skill/,
    );
    expect(() => interpolateObject("{{ missing }}", {}, "my-skill")).toThrow(
      /missing/i,
    );
  });

  // ── Arrays ──────────────────────────────────────────────────────

  it("interpolates strings in arrays", () => {
    const result = interpolateObject(
      ["{{ a }}", "plain", "{{ b }}"],
      { a: "X", b: "Y" },
      "test",
    );
    expect(result).toEqual(["X", "plain", "Y"]);
  });

  it("passes non-string array elements through", () => {
    const result = interpolateObject([1, true, null], {}, "test");
    expect(result).toEqual([1, true, null]);
  });

  // ── Objects ─────────────────────────────────────────────────────

  it("interpolates string values in objects", () => {
    const result = interpolateObject(
      { key: "{{ val }}", other: "static" },
      { val: "dynamic" },
      "test",
    );
    expect(result).toEqual({ key: "dynamic", other: "static" });
  });

  it("recursively interpolates nested objects", () => {
    const result = interpolateObject(
      {
        outer: {
          inner: "{{ x }}",
        },
      },
      { x: "deep" },
      "test",
    );
    expect(result).toEqual({ outer: { inner: "deep" } });
  });

  it("handles mixed nested structures", () => {
    const input = {
      list: ["{{ a }}", 42],
      nested: { val: "{{ b }}" },
      plain: "no-template",
    };
    const result = interpolateObject(input, { a: "A", b: "B" }, "test");
    expect(result).toEqual({
      list: ["A", 42],
      nested: { val: "B" },
      plain: "no-template",
    });
  });
});
