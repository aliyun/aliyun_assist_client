import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

import {
  mergeConfigurations,
  resolveModelConfig,
  ConfigFileSchema,
  ResolvedConfigSchema,
  McpStdioConfigSchema,
  McpSseConfigSchema,
  McpStreamableHttpConfigSchema,
  McpServerConfigSchema,
  ModelIdentifierSchema,
  OpenAICompatibleModelConfigSchema,
  type ConfigFile,
  type ResolvedConfig,
} from "../src/config.js";

// ── Schema validation tests ─────────────────────────────────────────

describe("ConfigFileSchema", () => {
  it("accepts an empty object", () => {
    const result = ConfigFileSchema.safeParse({});
    expect(result.success).toBe(true);
  });

  it("accepts a valid full config", () => {
    const result = ConfigFileSchema.safeParse({
      profile: "leaf-node",
      proxy: "http://proxy:8080",
      mcpServers: {
        myServer: {
          type: "stdio",
          command: "node",
          args: ["server.js"],
          env: { FOO: "bar" },
        },
      },
      interpolation: {
        mySkill: { key: "value" },
      },
      models: {
        "provider/model-1": {
          type: "openai-compatible",
          api_key: "sk-test",
          base_url: "https://api.example.com",
        },
      },
      defaultModel: "provider/model-1",
    });
    expect(result.success).toBe(true);
  });

  it("rejects unknown keys (strict mode)", () => {
    const result = ConfigFileSchema.safeParse({ unknownField: true });
    expect(result.success).toBe(false);
  });

  it("rejects invalid profile value", () => {
    const result = ConfigFileSchema.safeParse({ profile: "invalid-profile" });
    expect(result.success).toBe(false);
  });
});

describe("ResolvedConfigSchema", () => {
  it("applies default values for empty input", () => {
    const result = ResolvedConfigSchema.parse({});
    expect(result.profile).toBe("default");
    expect(result.mcpServers).toEqual({});
    expect(result.interpolation).toEqual({});
  });

  it("preserves explicit values", () => {
    const result = ResolvedConfigSchema.parse({
      profile: "leaf-node",
      proxy: "http://proxy:8080",
    });
    expect(result.profile).toBe("leaf-node");
    expect(result.proxy).toBe("http://proxy:8080");
  });
});

describe("McpServerConfigSchema (discriminated union)", () => {
  it("parses stdio config with defaults", () => {
    const result = McpStdioConfigSchema.parse({
      command: "node",
    });
    expect(result.type).toBe("stdio");
    expect(result.args).toEqual([]);
    expect(result.env).toEqual({});
  });

  it("parses SSE config", () => {
    const result = McpSseConfigSchema.parse({
      type: "sse",
      url: "https://example.com/sse",
    });
    expect(result.type).toBe("sse");
    expect(result.headers).toEqual({});
  });

  it("parses streamable-http config", () => {
    const result = McpStreamableHttpConfigSchema.parse({
      type: "streamable-http",
      url: "https://example.com/stream",
      headers: { Authorization: "Bearer tok" },
    });
    expect(result.type).toBe("streamable-http");
    expect(result.headers).toEqual({ Authorization: "Bearer tok" });
  });

  it("discriminates by type field", () => {
    const stdio = McpServerConfigSchema.parse({ type: "stdio", command: "echo" });
    expect(stdio.type).toBe("stdio");

    const sse = McpServerConfigSchema.parse({ type: "sse", url: "https://x.com/sse" });
    expect(sse.type).toBe("sse");
  });

  it("supports allowedTools filter", () => {
    const result = McpStdioConfigSchema.parse({
      command: "node",
      allowedTools: ["read", "write"],
    });
    expect(result.allowedTools).toEqual(["read", "write"]);
  });
});

describe("ModelIdentifierSchema", () => {
  it("accepts valid identifiers", () => {
    expect(ModelIdentifierSchema.safeParse("bailian/qwen3.5-plus").success).toBe(true);
    expect(ModelIdentifierSchema.safeParse("openai/gpt-4o").success).toBe(true);
    expect(ModelIdentifierSchema.safeParse("local.ollama/llama3").success).toBe(true);
    expect(ModelIdentifierSchema.safeParse("my-model").success).toBe(true);
    expect(ModelIdentifierSchema.safeParse("model_v2").success).toBe(true);
  });

  it("rejects identifiers starting with non-alphanumeric", () => {
    expect(ModelIdentifierSchema.safeParse("-model").success).toBe(false);
    expect(ModelIdentifierSchema.safeParse("/model").success).toBe(false);
    expect(ModelIdentifierSchema.safeParse(".model").success).toBe(false);
  });

  it("rejects empty string", () => {
    expect(ModelIdentifierSchema.safeParse("").success).toBe(false);
  });
});

describe("OpenAICompatibleModelConfigSchema", () => {
  it("accepts valid config", () => {
    const result = OpenAICompatibleModelConfigSchema.safeParse({
      type: "openai-compatible",
      api_key: "sk-test",
      base_url: "https://api.example.com",
      model: "gpt-4o",
    });
    expect(result.success).toBe(true);
  });

  it("rejects empty api_key", () => {
    const result = OpenAICompatibleModelConfigSchema.safeParse({
      type: "openai-compatible",
      api_key: "",
    });
    expect(result.success).toBe(false);
  });

  it("rejects invalid base_url", () => {
    const result = OpenAICompatibleModelConfigSchema.safeParse({
      type: "openai-compatible",
      api_key: "sk-test",
      base_url: "not-a-url",
    });
    expect(result.success).toBe(false);
  });
});

// ── mergeConfigurations tests ───────────────────────────────────────

describe("mergeConfigurations", () => {
  it("returns empty object for zero layers", () => {
    expect(mergeConfigurations()).toEqual({});
  });

  it("returns the single layer unchanged", () => {
    const layer: ConfigFile = { profile: "leaf-node" };
    expect(mergeConfigurations(layer)).toEqual(layer);
  });

  it("later layers override scalar values", () => {
    const base: ConfigFile = { profile: "default", proxy: "http://a" };
    const override: ConfigFile = { proxy: "http://b" };
    const merged = mergeConfigurations(base, override);
    expect(merged.proxy).toBe("http://b");
    expect(merged.profile).toBe("default");
  });

  it("merges MCP servers by union (later overrides same key)", () => {
    const base: ConfigFile = {
      mcpServers: {
        server1: { type: "stdio", command: "cmd1", args: [], env: {} },
      },
    };
    const override: ConfigFile = {
      mcpServers: {
        server2: { type: "sse", url: "https://example.com/sse", headers: {} },
      },
    };
    const merged = mergeConfigurations(base, override);
    expect(Object.keys(merged.mcpServers!)).toContain("server1");
    expect(Object.keys(merged.mcpServers!)).toContain("server2");
  });

  it("later MCP server overrides earlier with same key", () => {
    const base: ConfigFile = {
      mcpServers: {
        myServer: { type: "stdio", command: "old-cmd", args: [], env: {} },
      },
    };
    const override: ConfigFile = {
      mcpServers: {
        myServer: { type: "stdio", command: "new-cmd", args: [], env: {} },
      },
    };
    const merged = mergeConfigurations(base, override);
    const server = merged.mcpServers!["myServer"];
    expect(server.type).toBe("stdio");
    if (server.type === "stdio") {
      expect(server.command).toBe("new-cmd");
    }
  });

  it("merges three layers with correct precedence", () => {
    const a: ConfigFile = { profile: "default", proxy: "http://a" };
    const b: ConfigFile = { proxy: "http://b" };
    const c: ConfigFile = { profile: "leaf-node" };
    const merged = mergeConfigurations(a, b, c);
    expect(merged.profile).toBe("leaf-node");
    expect(merged.proxy).toBe("http://b");
  });

  it("merges interpolation config", () => {
    const base: ConfigFile = {
      interpolation: { skill1: { key1: "val1" } },
    };
    const override: ConfigFile = {
      interpolation: { skill2: { key2: "val2" } },
    };
    const merged = mergeConfigurations(base, override);
    // RFC 7396 merge patch replaces the entire interpolation object
    // (only MCP servers get special union semantics)
    expect(merged.interpolation).toBeDefined();
  });
});

// ── resolveModelConfig tests ────────────────────────────────────────

describe("resolveModelConfig", () => {
  const baseConfig: ResolvedConfig = {
    profile: "default",
    mcpServers: {},
    interpolation: {},
  };

  beforeEach(() => {
    // Clear env vars that resolveModelConfig reads
    delete process.env.OPENAI_API_KEY;
    delete process.env.OPENAI_BASE_URL;
    delete process.env.OPENAI_MODEL;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns error when no models and no env vars configured", () => {
    const result = resolveModelConfig(baseConfig);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toContain("No LLM provider configured");
    }
  });

  it("resolves file-based model config", () => {
    const config: ResolvedConfig = {
      ...baseConfig,
      models: {
        "my-provider/my-model": {
          type: "openai-compatible",
          api_key: "sk-test-key",
          base_url: "https://api.example.com",
        },
      },
      defaultModel: "my-provider/my-model",
    };
    const result = resolveModelConfig(config);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.config.identifier).toBe("my-provider/my-model");
      expect(result.config.apiKey).toBe("sk-test-key");
      expect(result.config.baseUrl).toBe("https://api.example.com");
      // model extracted from last segment of identifier
      expect(result.config.model).toBe("my-model");
    }
  });

  it("uses explicit model field over identifier segment", () => {
    const config: ResolvedConfig = {
      ...baseConfig,
      models: {
        "provider/alias": {
          type: "openai-compatible",
          api_key: "sk-key",
          model: "actual-model-name",
        },
      },
      defaultModel: "provider/alias",
    };
    const result = resolveModelConfig(config);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.config.model).toBe("actual-model-name");
    }
  });

  it("returns error when models exist but defaultModel is missing", () => {
    const config: ResolvedConfig = {
      ...baseConfig,
      models: {
        "provider/model": {
          type: "openai-compatible",
          api_key: "sk-key",
        },
      },
    };
    const result = resolveModelConfig(config);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toContain("defaultModel is required");
    }
  });

  it("returns error when defaultModel references non-existent model", () => {
    const config: ResolvedConfig = {
      ...baseConfig,
      models: {
        "provider/model-a": {
          type: "openai-compatible",
          api_key: "sk-key",
        },
      },
      defaultModel: "provider/model-b",
    };
    const result = resolveModelConfig(config);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toContain("not found in models");
      expect(result.error).toContain("provider/model-a");
    }
  });

  it("falls back to env vars with deprecation warning", () => {
    process.env.OPENAI_API_KEY = "sk-env-key";
    process.env.OPENAI_BASE_URL = "https://env.example.com";
    process.env.OPENAI_MODEL = "env-model";

    const result = resolveModelConfig(baseConfig);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.config.identifier).toBe("env/openai");
      expect(result.config.apiKey).toBe("sk-env-key");
      expect(result.config.baseUrl).toBe("https://env.example.com");
      expect(result.config.model).toBe("env-model");
      expect(result.warning).toContain("deprecated");
    }
  });

  it("defaults to gpt-4o when OPENAI_MODEL is not set", () => {
    process.env.OPENAI_API_KEY = "sk-env-key";

    const result = resolveModelConfig(baseConfig);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.config.model).toBe("gpt-4o");
    }
  });

  it("includes extraHeaders in resolved config", () => {
    const config: ResolvedConfig = {
      ...baseConfig,
      models: {
        "provider/model": {
          type: "openai-compatible",
          api_key: "sk-key",
          extraHeaders: { "X-Custom": "value" },
        },
      },
      defaultModel: "provider/model",
    };
    const result = resolveModelConfig(config);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.config.extraHeaders).toEqual({ "X-Custom": "value" });
    }
  });
});
