import type { SessionUpdate } from "@agentclientprotocol/sdk";
import type { Notification } from "protocol";

/**
 * Maps a single internal Notification to an ACP SessionUpdate.
 * Returns undefined if the notification produces no ACP-visible output.
 */
export function toSessionUpdate(n: Notification): SessionUpdate | undefined {
  switch (n.method) {
    case "item/delta": {
      const { delta } = n.params;
      switch (delta.type) {
        case "agentMessage":
          return {
            sessionUpdate: "agent_message_chunk",
            content: { type: "text", text: delta.delta },
          };
        case "reasoning":
          return {
            sessionUpdate: "agent_thought_chunk",
            content: { type: "text", text: delta.delta },
          };
        case "commandOutput":
          return {
            sessionUpdate: "tool_call_update",
            toolCallId: n.params.itemId,
            content: [
              {
                type: "content",
                content: { type: "text", text: delta.delta },
              },
            ],
          };
      }
      break;
    }
    case "item/started": {
      const { item } = n.params;
      switch (item.type) {
        case "commandExecution":
          return {
            sessionUpdate: "tool_call",
            toolCallId: item.id,
            title: `run_shell: ${item.command}`,
            status: "in_progress",
            kind: "execute",
          };
        case "toolCall":
          return {
            sessionUpdate: "tool_call",
            toolCallId: item.id,
            title: `${item.toolName}`,
            status: "in_progress",
            kind: "other",
          };
        case "mcpToolCall":
          return {
            sessionUpdate: "tool_call",
            toolCallId: item.id,
            title: `${item.serverName}/${item.toolName}`,
            status: "in_progress",
            kind: "other",
          };
        case "agentMessage":
        case "reasoning":
          return undefined;
      }
      break;
    }
    case "item/completed": {
      const { item } = n.params;
      switch (item.type) {
        case "commandExecution":
          return {
            sessionUpdate: "tool_call_update",
            toolCallId: item.id,
            status: item.status === "failed" ? "failed" : "completed",
          };
        case "toolCall":
          return {
            sessionUpdate: "tool_call_update",
            toolCallId: item.id,
            status: item.status === "failed" ? "failed" : "completed",
          };
        case "mcpToolCall":
          return {
            sessionUpdate: "tool_call_update",
            toolCallId: item.id,
            status: item.status === "failed" ? "failed" : "completed",
          };
        case "agentMessage":
        case "reasoning":
          return undefined;
      }
      break;
    }
    case "turn/started":
    case "turn/completed":
    case "request/resolved":
    case "error":
      return undefined;
    default:
      return undefined;
  }
}
