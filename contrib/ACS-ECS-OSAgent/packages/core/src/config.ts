import { readFileSync, existsSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

import * as yaml from 'js-yaml';
import { apply as mergePatch } from 'json-merge-patch';
import { z } from 'zod';

export const AgentProfileSchema = z.enum(['default', 'leaf-node']);
export type AgentProfile = z.infer<typeof AgentProfileSchema>;

// MCP transport type discriminator
export type McpTransportType = 'stdio' | 'sse' | 'streamable-http';

// Shared fields for all MCP server configurations
const McpServerBaseSchema = z.object({
  allowedTools: z.array(z.string()).optional(),
});

// Stdio transport config
export const McpStdioConfigSchema = McpServerBaseSchema.extend({
  type: z.literal('stdio').optional().default('stdio'),
  command: z.string(),
  args: z.array(z.string()).optional().default([]),
  env: z.record(z.string()).optional().default({}),
});

// SSE transport config
export const McpSseConfigSchema = McpServerBaseSchema.extend({
  type: z.literal('sse'),
  url: z.string().url(),
  headers: z.record(z.string()).optional().default({}),
});

// Streamable HTTP transport config
export const McpStreamableHttpConfigSchema = McpServerBaseSchema.extend({
  type: z.literal('streamable-http'),
  url: z.string().url(),
  headers: z.record(z.string()).optional().default({}),
});

// Union schema for all transport types
export const McpServerConfigSchema = z.discriminatedUnion('type', [
  McpStdioConfigSchema,
  McpSseConfigSchema,
  McpStreamableHttpConfigSchema,
]);

export type McpServerConfig = z.infer<typeof McpServerConfigSchema>;

// Interpolation config: skill-scoped key-value pairs for template substitution
export const InterpolationConfigSchema = z.record(z.record(z.string()));
export type InterpolationConfig = z.infer<typeof InterpolationConfigSchema>;

// ============================================================================
// Model Provider Configuration
// ============================================================================

// Model identifier: alphanumeric, slash, dot, hyphen, underscore
// Examples: bailian/qwen3.5-plus, openai/gpt-4o, local.ollama/llama3
export const ModelIdentifierSchema = z.string().regex(
  /^[a-zA-Z0-9][a-zA-Z0-9/._-]*$/,
  'Model identifier must start with alphanumeric and contain only alphanumeric, /, ., -, _'
);
export type ModelIdentifier = z.infer<typeof ModelIdentifierSchema>;

// OpenAI-compatible provider configuration
export const OpenAICompatibleModelConfigSchema = z.object({
  type: z.literal('openai-compatible'),
  api_key: z.string().min(1, 'api_key is required'),
  base_url: z.string().url().optional(),
  model: z.string().optional(),
  extraHeaders: z.record(z.string()).optional(),
});
export type OpenAICompatibleModelConfig = z.infer<typeof OpenAICompatibleModelConfigSchema>;

// Union schema for all model provider types (currently only openai-compatible)
export const ModelConfigSchema = z.discriminatedUnion('type', [
  OpenAICompatibleModelConfigSchema,
]);
export type ModelConfig = z.infer<typeof ModelConfigSchema>;

// Models map: identifier -> config
export const ModelsConfigSchema = z.record(ModelIdentifierSchema, ModelConfigSchema);
export type ModelsConfig = z.infer<typeof ModelsConfigSchema>;

// Resolved model config for runtime use
export type ResolvedModelConfig = {
  identifier: string;
  apiKey: string;
  baseUrl?: string;
  model: string;
  extraHeaders?: Record<string, string>;
};

// Model config resolution result
export type ModelConfigResult =
  | { ok: true; config: ResolvedModelConfig; warning?: string }
  | { ok: false; error: string };

export const ConfigFileSchema = z.object({
  profile: AgentProfileSchema.optional(),
  proxy: z.string().optional(),
  mcpServers: z.record(McpServerConfigSchema).optional(),
  interpolation: InterpolationConfigSchema.optional(),
  // Model provider configuration
  models: ModelsConfigSchema.optional(),
  defaultModel: ModelIdentifierSchema.optional(),
}).strict(); // Reject unknown keys at validation time

export type ConfigFile = z.infer<typeof ConfigFileSchema>;

export const ResolvedConfigSchema = z.object({
  profile: AgentProfileSchema.default('default'),
  proxy: z.string().optional(),
  mcpServers: z.record(McpServerConfigSchema).default({}),
  interpolation: InterpolationConfigSchema.default({}),
  // Model configuration (either from file or env var fallback)
  models: ModelsConfigSchema.optional(),
  defaultModel: ModelIdentifierSchema.optional(),
});

export type ResolvedConfig = z.infer<typeof ResolvedConfigSchema>;

/**
 * Merges multiple config layers using RFC 7396 JSON Merge Patch.
 *
 * @param layers - Config objects in order of increasing priority
 *                 (later layers override earlier ones)
 * @returns Merged config
 */
export function mergeConfigurations(...layers: ConfigFile[]): ConfigFile {
  if (layers.length === 0) return {};
  if (layers.length === 1) return layers[0];

  // Base merge: RFC 7396 (later layers patch earlier ones)
  let result = layers[0];
  for (let i = 1; i < layers.length; i++) {
    result = mergePatch(result, layers[i]) as ConfigFile;
  }

  // MCP servers: union by key (merge server definitions)
  const mcpServerArrays = layers
    .map(l => l.mcpServers)
    .filter((servers): servers is Record<string, McpServerConfig> => 
      servers !== undefined && Object.keys(servers).length > 0
    );

  if (mcpServerArrays.length > 1) {
    // Deep merge: later layers override earlier ones for same server name
    result.mcpServers = mcpServerArrays.reduce((acc, servers) => ({
      ...acc,
      ...servers,
    }), {});
  }

  return result;
}

function parseAndValidateConfig(path: string): ConfigFile | null {
  if (!existsSync(path)) return null;

  try {
    const content = readFileSync(path, 'utf-8');
    const raw = yaml.load(content);

    // Zod validation — throws on unknown keys or type mismatch
    const result = ConfigFileSchema.parse(raw);
    return result;
  } catch (err) {
    if (err instanceof z.ZodError) {
      const issues = err.issues.map(i => `${i.path.join('.')}: ${i.message}`).join('; ');
      console.warn(`[config] Schema validation failed in ${path}: ${issues}, skipping.`);
    } else {
      console.warn(`[config] Failed to parse ${path}: ${err instanceof Error ? err.message : String(err)}, skipping.`);
    }
    return null;
  }
}

interface ConfigPaths {
  user: string;
  userLegacy: string;
  project: string;
}

function getConfigPaths(cwd: string): ConfigPaths {
  return {
    user: join(homedir(), '.osagent', 'config.yaml'),
    userLegacy: join(homedir(), '.config', 'osagent', 'config.yaml'),
    project: join(cwd, '.osagent.yaml'),
  };
}

/**
 * Resolves model configuration from config or environment variables.
 * Priority: file-based config > env var fallback (legacy)
 */
export function resolveModelConfig(config: ResolvedConfig): ModelConfigResult {
  const { models, defaultModel } = config;

  // Case 1: File-based configuration (recommended)
  if (models && Object.keys(models).length > 0) {
    if (!defaultModel) {
      return {
        ok: false,
        error: 'defaultModel is required when models are configured.',
      };
    }

    const modelConfig = models[defaultModel];
    if (!modelConfig) {
      const available = Object.keys(models).join(', ');
      return {
        ok: false,
        error: `Model "${defaultModel}" not found in models. Available: ${available}`,
      };
    }

    // Extract the model name: use explicit model field, or last segment of identifier
    const modelName = modelConfig.model ?? defaultModel.split('/').pop() ?? defaultModel;

    return {
      ok: true,
      config: {
        identifier: defaultModel,
        apiKey: modelConfig.api_key,
        baseUrl: modelConfig.base_url,
        model: modelName,
        extraHeaders: modelConfig.extraHeaders,
      },
    };
  }

  // Case 2: Environment variable fallback (legacy, not recommended)
  const envApiKey = process.env.OPENAI_API_KEY;
  if (envApiKey) {
    return {
      ok: true,
      config: {
        identifier: 'env/openai',
        apiKey: envApiKey,
        baseUrl: process.env.OPENAI_BASE_URL,
        model: process.env.OPENAI_MODEL ?? 'gpt-4o',
      },
      warning: `Using environment variables for LLM configuration is deprecated.

Environment variables are inherited by child processes, which poses a security risk.
Please migrate to file-based configuration in ~/.config/osagent/config.yaml:

  models:
    my-model:
      type: openai-compatible
      api_key: <your-api-key>
      base_url: <optional-base-url>

  defaultModel: my-model`,
    };
  }

  // Case 3: No configuration
  return {
    ok: false,
    error: `No LLM provider configured.

Please configure at least one model in ~/.osagent/config.yaml:

  models:
    my-model:
      type: openai-compatible
      api_key: <your-api-key>
      base_url: <optional-base-url>

  defaultModel: my-model`,
  };
}

export function loadConfig(cwd: string = process.cwd()): ResolvedConfig {
  const paths = getConfigPaths(cwd);

  // 1. Parse + validate user-level config with migration handling
  let userConfig: ConfigFile = {};
  const newUserConfigExists = existsSync(paths.user);
  const legacyUserConfigExists = existsSync(paths.userLegacy);

  if (newUserConfigExists) {
    // New config exists - use it
    userConfig = parseAndValidateConfig(paths.user) ?? {};
    if (legacyUserConfigExists) {
      // Warn that legacy config is being ignored
      console.warn(
        `[config] Found legacy config at ${paths.userLegacy}. ` +
        `Using ${paths.user} instead. Please remove the legacy file.`
      );
    }
  } else if (legacyUserConfigExists) {
    // Only legacy config exists - use it with migration warning
    userConfig = parseAndValidateConfig(paths.userLegacy) ?? {};
    console.warn(
      `[config] Using legacy config at ${paths.userLegacy}. ` +
      `Please migrate to ${paths.user}.`
    );
  }

  // 2. Parse + validate project-level config
  const projectConfig = parseAndValidateConfig(paths.project) ?? {};

  // 3. Merge layers with Kubernetes-style semantics
  const merged = mergeConfigurations(userConfig, projectConfig);

  // 4. Apply env-var overrides (higher priority than files)
  const envProfile = process.env.OSAGENT_PROFILE;
  if (envProfile) {
    const parsedProfile = AgentProfileSchema.safeParse(envProfile);
    if (parsedProfile.success) {
      merged.profile = parsedProfile.data;
    }
  }

  const envProxy = process.env.HTTPS_PROXY ?? process.env.https_proxy
    ?? process.env.HTTP_PROXY ?? process.env.http_proxy;
  if (envProxy) {
    merged.proxy = envProxy;
  }

  // 5. Return validated result with built-in defaults applied by Zod
  return ResolvedConfigSchema.parse(merged);
}
