import type { ChatCompletionMessageParam } from "openai/resources/chat/completions";

/** A composable section of the conversation context window. */
export interface ContextSection {
  /** Build the message(s) for this section. */
  build(): ChatCompletionMessageParam[];
}
