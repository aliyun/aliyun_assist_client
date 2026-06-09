import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { SSEClientTransport } from '@modelcontextprotocol/sdk/client/sse.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';

import type { McpServerConfig } from './config.js';

export interface McpTool {
  server: string;
  originalName: string;  // Original tool name from MCP server
  mangledName: string;   // LLM-safe name (≤64 chars)
  definition: {
    name: string;        // Same as mangledName
    description: string;
    input_schema: object;
  };
}

export class McpService {
  private clients = new Map<string, Client>();
  private tools: McpTool[] = [];
  private connectedServers = new Set<string>();

  /**
   * Initialize MCP service with a set of servers.
   * This is typically called at thread start with global MCP servers.
   */
  async initialize(servers: Record<string, McpServerConfig>): Promise<void> {
    for (const [name, config] of Object.entries(servers)) {
      await this.addServer(name, config);
    }
  }

  /**
   * Dynamically add an MCP server.
   * Called when a skill with MCP dependencies is activated.
   * @param name - Server name (e.g., "filesystem")
   * @param config - Server configuration
   * @param skillName - If provided, server is skill-sourced and namespaced accordingly
   * @returns true if the server was added, false if already connected
   */
  async addServer(name: string, config: McpServerConfig, skillName?: string): Promise<boolean> {
    // Internal key: namespace with skill name to avoid collisions
    const serverKey = skillName ? `${skillName}_${name}` : name;

    // Skip if already connected
    if (this.connectedServers.has(serverKey)) {
      return false;
    }

    let transport;

    // Create transport based on type
    switch (config.type) {
      case 'sse':
        transport = new SSEClientTransport(new URL(config.url), {
          requestInit: { headers: config.headers },
        });
        break;

      case 'streamable-http':
        transport = new StreamableHTTPClientTransport(new URL(config.url), {
          requestInit: { headers: config.headers },
        });
        break;

      case 'stdio':
      default:
        // Filter out undefined values from process.env
        const filteredEnv: Record<string, string> = {};
        for (const [key, value] of Object.entries(process.env)) {
          if (value !== undefined) {
            filteredEnv[key] = value;
          }
        }
        transport = new StdioClientTransport({
          command: config.command,
          args: config.args,
          env: { ...filteredEnv, ...config.env },
          stderr: 'pipe',
        });
        break;
    }

    const client = new Client(
      { name: 'osagent', version: '0.0.1' },
      { capabilities: {} }
    );

    await client.connect(transport);

    // List available tools from this server
    const toolsResponse = await client.listTools();
    const allowedTools = config.allowedTools;
    for (const tool of toolsResponse.tools) {
      // Filter tools based on allowedTools configuration
      if (allowedTools !== undefined && allowedTools.length > 0) {
        if (!allowedTools.includes(tool.name)) {
          continue; // Skip tools not in the allowed list
        }
      }

      // Generate LLM-safe tool name (≤64 chars)
      // Format: skill_<skill>_mcp_<server>_<tool> for skill servers, mcp_<server>_<tool> for global
      const baseName = skillName
        ? `skill_${skillName}_mcp_${name}_${tool.name}`
        : `mcp_${name}_${tool.name}`;
      let mangledName: string;

      if (baseName.length <= 64) {
        mangledName = baseName;
      } else {
        // Use hash truncation with prefix-aware length enforcement
        const hash = await this.hashString(tool.name);
        const prefix = skillName
          ? `skill_${skillName}_mcp_${name}_`
          : `mcp_${name}_`;
        const maxHashLen = Math.max(1, 64 - prefix.length);
        mangledName = `${prefix}${hash.slice(0, maxHashLen)}`;
      }

      this.tools.push({
        server: serverKey,
        originalName: tool.name,
        mangledName,
        definition: {
          name: mangledName,
          description: tool.description ?? '',
          input_schema: tool.inputSchema,
        },
      });
    }

    this.clients.set(serverKey, client);
    this.connectedServers.add(serverKey);
    return true;
  }

  getTools(): McpTool[] {
    return this.tools;
  }

  async callTool(server: string, mangledName: string, args: unknown): Promise<unknown> {
    const client = this.clients.get(server);
    if (!client) {
      throw new Error(`MCP server not found: ${server}`);
    }

    // Find original tool name from mangled name
    const tool = this.tools.find(t => t.server === server && t.mangledName === mangledName);
    if (!tool) {
      throw new Error(`MCP tool not found: ${mangledName} on server ${server}`);
    }

    const result = await client.callTool({ name: tool.originalName, arguments: args as { [x: string]: unknown } | undefined });
    return result;
  }

  /**
   * Generate a short hash for name mangling.
   * Returns first 8 chars of SHA-256 hex digest for collision resistance.
   */
  private async hashString(input: string): Promise<string> {
    const crypto = await import('node:crypto');
    return crypto.createHash('sha256').update(input).digest('hex').slice(0, 8);
  }

  async shutdown(): Promise<void> {
    for (const client of this.clients.values()) {
      try {
        await client.close();
      } catch {
        // Best-effort — continue closing remaining clients
      }
    }
    this.clients.clear();
    this.tools = [];
    this.connectedServers.clear();
  }
}
