/**
 * Color palette and spacing constants for the Ink TUI.
 */

export const COLORS = {
  /** User input text */
  userInput: "cyan",
  /** Assistant / agent message text */
  assistantText: "white",
  /** Tool call labels and borders */
  toolCall: "yellow",
  /** Tool call dim decorations */
  toolCallDim: "gray",
  /** Error messages */
  error: "red",
  /** Spinner indicator */
  spinner: "blue",
  /** Border / separator lines */
  border: "gray",
  /** Prompt prefix character */
  prompt: "green",
  /** Approval highlights */
  approval: "magenta",
  /** Approval key bindings */
  approvalKey: "cyan",
} as const;

/** Markdown rendering colors. */
export const MD_COLORS = {
  /** Heading level 1 */
  h1: "magentaBright",
  /** Heading level 2 */
  h2: "cyanBright",
  /** Heading level 3+ */
  h3: "blueBright",
  /** Inline code */
  inlineCode: "green",
  /** Code block border */
  codeBlockBorder: "gray",
  /** Blockquote bar and text */
  blockquote: "gray",
  /** Link URL portion */
  linkUrl: "gray",
  /** Link text (underlined separately) */
  linkText: "cyan",
  /** Horizontal rule */
  hr: "gray",
  /** List bullet / number */
  listMarker: "yellow",
} as const;

/** Status-based colors for command execution display. */
export const STATUS_COLORS = {
  pending: "yellow",
  running: "blue",
  success: "green",
  error: "red",
  denied: "gray",
} as const;

/** Tool-type accent colors for distinguishing tool call origins. */
export const TOOL_TYPE_COLORS = {
  /** Skill activation badges */
  skill: "magenta",
  /** MCP tool calls */
  mcp: "cyan",
} as const;

/** Column-0 icons for tool call types. */
export const TOOL_ICONS = {
  shell: "▶",
  skill: "★",
  mcp: "◆",
  denied: "⊘",
} as const;

export const SYMBOLS = {
  /** Prompt prefix shown before user input */
  promptPrefix: "> ",
} as const;

/** GFM Alert type colors. */
export const ALERT_COLORS = {
  NOTE: "blue",
  TIP: "green",
  IMPORTANT: "magenta",
  WARNING: "yellow",
  CAUTION: "red",
} as const;

/** GFM Alert type labels (text-only; colored left-bar provides visual distinction). */
export const ALERT_LABELS = {
  NOTE: "Note",
  TIP: "Tip",
  IMPORTANT: "Important",
  WARNING: "Warning",
  CAUTION: "Caution",
} as const;
