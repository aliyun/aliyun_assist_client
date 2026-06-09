import { HttpsProxyAgent } from "https-proxy-agent";
import OpenAI from "openai";
import type {
  ChatCompletionMessageParam,
  ChatCompletionTool,
} from "openai/resources/chat/completions";

import type { ResolvedModelConfig } from "./config.js";
import type { ContextSection } from "./context/index.js";
import {
  SYSTEM_SECTION,
  SKILLS_SECTION,
  createEnvironmentSection,
} from "./context/index.js";

export type ChatEvent =
  | { type: "reasoningStarted" }
  | { type: "reasoning"; delta: string }
  | { type: "reasoningCompleted"; summary: string[]; content: string[] }
  | { type: "text"; delta: string }
  | { type: "toolCall"; toolCall: { id: string; name: string; arguments: string } };

export class Conversation {
  private client: OpenAI;
  private model: string;
  private messages: ChatCompletionMessageParam[];

  constructor(modelConfig: ResolvedModelConfig, proxy?: string, cwd?: string) {
    this.client = new OpenAI({
      apiKey: modelConfig.apiKey,
      baseURL: modelConfig.baseUrl,
      defaultHeaders: modelConfig.extraHeaders,
      httpAgent: proxy ? new HttpsProxyAgent(proxy) : undefined,
    });
    this.model = modelConfig.model;
    const sections: ContextSection[] = [
      SYSTEM_SECTION,
      SKILLS_SECTION,
      createEnvironmentSection(cwd ?? process.cwd()),
    ];
    this.messages = sections.flatMap((s) => s.build());
  }

  async *chat(
    userMessage?: string,
    tools?: ChatCompletionTool[],
  ): AsyncGenerator<ChatEvent, void> {
    if (userMessage) {
      this.messages.push({ role: "user", content: userMessage });
    }

    try {
    const stream = await this.client.chat.completions.create({
      model: this.model,
      messages: this.messages,
      ...(tools?.length ? { tools } : {}),
      stream: true,
    });

    let content = "";
    let reasoningContent = "";
    const toolCalls = new Map<number, { id: string; name: string; arguments: string }>();
    let hasEmittedReasoningStarted = false;

    for await (const chunk of stream) {
      const choice = chunk.choices[0];
      if (!choice) continue;

      // Reasoning content delta (for models like o1 that support reasoning)
      // OpenAI SDK may expose this as a separate field
      const reasoningDelta = (choice.delta as unknown as { reasoning_content?: string }).reasoning_content;
      if (reasoningDelta) {
        if (!hasEmittedReasoningStarted) {
          hasEmittedReasoningStarted = true;
          yield { type: "reasoningStarted" };
        }
        reasoningContent += reasoningDelta;
        yield { type: "reasoning", delta: reasoningDelta };
      }

      // Text content delta
      const textDelta = choice.delta.content;
      if (textDelta) {
        // If we were reasoning, emit completion before text starts
        if (hasEmittedReasoningStarted) {
          const reasoningLines = reasoningContent.split("\n").filter(l => l.trim());
          yield { type: "reasoningCompleted", summary: reasoningLines.slice(0, 3), content: reasoningLines };
          hasEmittedReasoningStarted = false;
        }
        content += textDelta;
        yield { type: "text", delta: textDelta };
      }

      // Tool call deltas — accumulate fragments by index
      if (choice.delta.tool_calls) {
        for (const tc of choice.delta.tool_calls) {
          let entry = toolCalls.get(tc.index);
          if (!entry) {
            entry = { id: "", name: "", arguments: "" };
            toolCalls.set(tc.index, entry);
          }
          if (tc.id) entry.id = tc.id;
          if (tc.function?.name) entry.name = tc.function.name;
          if (tc.function?.arguments) entry.arguments += tc.function.arguments;
        }
      }
    }

    // Emit reasoningCompleted if still in reasoning state (no text was emitted)
    if (hasEmittedReasoningStarted) {
      const reasoningLines = reasoningContent.split("\n").filter(l => l.trim());
      yield { type: "reasoningCompleted", summary: reasoningLines.slice(0, 3), content: reasoningLines };
    }

    if (toolCalls.size > 0) {
      // Push assistant message with tool_calls array
      this.messages.push({
        role: "assistant",
        content: content || null,
        tool_calls: [...toolCalls.values()].map((tc) => ({
          id: tc.id,
          type: "function" as const,
          function: { name: tc.name, arguments: tc.arguments },
        })),
      });

      // Yield each assembled tool call
      for (const tc of toolCalls.values()) {
        yield { type: "toolCall", toolCall: tc };
      }
    } else {
      this.messages.push({ role: "assistant", content });
    }
    } catch (error) {
      // Roll back user message — no assistant reply was recorded,
      // so the conversation history stays consistent for retry.
      if (userMessage) {
        this.messages.pop();
      }
      throw error;
    }
  }

  addToolResult(toolCallId: string, content: string): void {
    this.messages.push({ role: "tool", tool_call_id: toolCallId, content });
  }
}
