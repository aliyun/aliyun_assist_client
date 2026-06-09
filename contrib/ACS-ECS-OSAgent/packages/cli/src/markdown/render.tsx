import React from "react";
import { Box, Text } from "ink";
import { Lexer } from "marked";
import type { Token, Tokens } from "marked";

import { ALERT_COLORS, ALERT_LABELS, MD_COLORS } from "../theme.js";
import { highlightCode } from "./highlight.js";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const HR_WIDTH = 40;
const HR_LINE = "─".repeat(HR_WIDTH);

/** Supported GFM Alert types. */
type GfmAlertType = keyof typeof ALERT_COLORS;

const ALERT_PATTERN = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*/;

/**
 * Detect if a blockquote is a GFM Alert. If the first paragraph starts with
 * `[!TYPE]`, extract the alert type and remaining content tokens.
 */
function detectGfmAlert(
  blockquote: Tokens.Blockquote,
): { alertType: GfmAlertType; contentTokens: Token[] } | null {
  if (blockquote.tokens.length === 0) return null;
  const first = blockquote.tokens[0];
  if (!first || first.type !== "paragraph") return null;

  const para = first as Tokens.Paragraph;
  const raw = para.text;
  const match = ALERT_PATTERN.exec(raw);
  if (!match) return null;

  const alertType = match[1] as GfmAlertType;
  // Strip the [!TYPE] prefix from the first paragraph
  const remainingText = raw.slice(match[0].length);

  const contentTokens: Token[] = [];
  if (remainingText.trim()) {
    // Re-lex the remaining text as a standalone paragraph
    const reLexed = Lexer.lex(remainingText);
    contentTokens.push(...reLexed);
  }
  // Include the rest of the blockquote tokens after the first paragraph
  contentTokens.push(...blockquote.tokens.slice(1));

  return { alertType, contentTokens };
}

// ---------------------------------------------------------------------------
// Heading color helper
// ---------------------------------------------------------------------------

function headingColor(depth: number): string {
  if (depth === 1) return MD_COLORS.h1;
  if (depth === 2) return MD_COLORS.h2;
  return MD_COLORS.h3;
}

// ---------------------------------------------------------------------------
// Inline token renderer
// ---------------------------------------------------------------------------

/**
 * Render an array of inline tokens into a flat array of `<Text>` elements.
 * These are meant to be composed inside a parent `<Text>` (Ink allows nested
 * `<Text>` but NOT `<Box>` inside `<Text>`).
 */
function renderInline(tokens: Token[]): React.ReactNode[] {
  const nodes: React.ReactNode[] = [];

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]!;

    switch (token.type) {
      case "text": {
        const t = token as Tokens.Text;
        // Text tokens can have nested tokens (e.g. inside list items)
        if (t.tokens && t.tokens.length > 0) {
          nodes.push(...renderInline(t.tokens));
        } else {
          nodes.push(<Text key={i}>{t.text}</Text>);
        }
        break;
      }

      case "strong": {
        const t = token as Tokens.Strong;
        nodes.push(
          <Text key={i} bold>
            {renderInline(t.tokens)}
          </Text>,
        );
        break;
      }

      case "em": {
        const t = token as Tokens.Em;
        nodes.push(
          <Text key={i} italic>
            {renderInline(t.tokens)}
          </Text>,
        );
        break;
      }

      case "codespan": {
        const t = token as Tokens.Codespan;
        nodes.push(
          <Text key={i} color={MD_COLORS.inlineCode}>
            {"`"}{t.text}{"`"}
          </Text>,
        );
        break;
      }

      case "link": {
        const t = token as Tokens.Link;
        nodes.push(
          <Text key={i}>
            <Text underline color={MD_COLORS.linkText}>
              {t.text}
            </Text>
            <Text dimColor color={MD_COLORS.linkUrl}>
              {" "}({t.href})
            </Text>
          </Text>,
        );
        break;
      }

      case "del": {
        const t = token as Tokens.Del;
        nodes.push(
          <Text key={i} strikethrough>
            {renderInline(t.tokens)}
          </Text>,
        );
        break;
      }

      case "br":
        nodes.push(<Text key={i}>{"\n"}</Text>);
        break;

      case "escape": {
        const t = token as Tokens.Escape;
        nodes.push(<Text key={i}>{t.text}</Text>);
        break;
      }

      case "image": {
        const t = token as Tokens.Image;
        nodes.push(
          <Text key={i} dimColor>
            [image: {t.text}]
          </Text>,
        );
        break;
      }

      case "html": {
        const t = token as Tokens.HTML;
        nodes.push(<Text key={i}>{t.text}</Text>);
        break;
      }

      default:
        // Fallback: render raw text for any unrecognised inline token
        nodes.push(<Text key={i}>{token.raw}</Text>);
        break;
    }
  }

  return nodes;
}

// ---------------------------------------------------------------------------
// Block token renderer
// ---------------------------------------------------------------------------

/**
 * Render an array of block-level tokens into Ink-compatible React elements.
 * Each block is wrapped in a `<Box>` so elements can stack vertically.
 */
function renderBlocks(tokens: Token[]): React.ReactNode[] {
  const nodes: React.ReactNode[] = [];

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]!;

    switch (token.type) {
      case "heading": {
        const t = token as Tokens.Heading;
        const prefix = "#".repeat(t.depth) + " ";
        nodes.push(
          <Box key={i}>
            <Text bold color={headingColor(t.depth)}>
              {prefix}{renderInline(t.tokens)}
            </Text>
          </Box>,
        );
        break;
      }

      case "paragraph": {
        const t = token as Tokens.Paragraph;
        nodes.push(
          <Box key={i}>
            <Text>{renderInline(t.tokens)}</Text>
          </Box>,
        );
        break;
      }

      case "code": {
        const t = token as Tokens.Code;
        const langLabel = t.lang ? ` ${t.lang} ` : "";
        nodes.push(
          <Box
            key={i}
            flexDirection="column"
            borderStyle="single"
            borderColor={MD_COLORS.codeBlockBorder}
            paddingLeft={1}
            paddingRight={1}
          >
            {langLabel && (
              <Text dimColor>{langLabel}</Text>
            )}
            <Text>{highlightCode(t.text, t.lang || undefined)}</Text>
          </Box>,
        );
        break;
      }

      case "blockquote": {
        const t = token as Tokens.Blockquote;
        // Check for GFM Alert syntax: first paragraph starts with [!TYPE]
        const alertInfo = detectGfmAlert(t);
        if (alertInfo) {
          const { alertType, contentTokens } = alertInfo;
          const color = ALERT_COLORS[alertType];
          const title = ALERT_LABELS[alertType];
          const inner = renderBlocks(contentTokens);
          nodes.push(
            <Box key={i} flexDirection="column">
              <Box flexDirection="row">
                <Text color={color}>│</Text>
                <Text> </Text>
                <Text color={color} bold>{title}</Text>
              </Box>
              {inner.map((child, ci) => (
                <Box key={ci} flexDirection="row">
                  <Text color={color}>│</Text>
                  <Text> </Text>
                  <Box flexDirection="column">{child}</Box>
                </Box>
              ))}
            </Box>,
          );
        } else {
          const inner = renderBlocks(t.tokens);
          nodes.push(
            <Box key={i} flexDirection="column">
              {inner.map((child, ci) => (
                <Box key={ci} flexDirection="row">
                  <Text color={MD_COLORS.blockquote}>│</Text>
                  <Text> </Text>
                  <Box flexDirection="column">{child}</Box>
                </Box>
              ))}
            </Box>,
          );
        }
        break;
      }

      case "list": {
        const t = token as Tokens.List;
        nodes.push(
          <Box key={i} flexDirection="column">
            {t.items.map((item, idx) => {
              const marker = t.ordered
                ? `${(typeof t.start === "number" ? t.start : 1) + idx}. `
                : "• ";
              return (
                <Box key={idx} flexDirection="row">
                  <Text color={MD_COLORS.listMarker}>{"  "}{marker}</Text>
                  <Box flexDirection="column">
                    {renderBlocks(item.tokens)}
                  </Box>
                </Box>
              );
            })}
          </Box>,
        );
        break;
      }

      case "hr":
        nodes.push(
          <Box key={i}>
            <Text dimColor color={MD_COLORS.hr}>{HR_LINE}</Text>
          </Box>,
        );
        break;

      case "table": {
        const t = token as Tokens.Table;
        // Render a minimal plain-text table
        const headerCells = t.header.map((cell) =>
          cell.tokens.map((tok) => tok.raw).join(""),
        );
        const separator = headerCells.map((h) => "-".repeat(h.length || 3));
        const rows = t.rows.map((row) =>
          row.map((cell) => cell.tokens.map((tok) => tok.raw).join("")),
        );
        nodes.push(
          <Box key={i} flexDirection="column">
            <Text>{headerCells.join(" | ")}</Text>
            <Text>{separator.join(" | ")}</Text>
            {rows.map((row, ri) => (
              <Text key={ri}>{row.join(" | ")}</Text>
            ))}
          </Box>,
        );
        break;
      }

      case "html": {
        const t = token as Tokens.HTML;
        nodes.push(
          <Box key={i}>
            <Text>{t.text}</Text>
          </Box>,
        );
        break;
      }

      case "space":
        // Blank lines — skip, spacing handled by marginBottom on blocks
        break;

      case "def":
        // Link reference definitions — not rendered visually
        break;

      default:
        // Fallback: render raw text for unrecognised block tokens
        nodes.push(
          <Box key={i}>
            <Text>{token.raw}</Text>
          </Box>,
        );
        break;
    }
  }

  return nodes;
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Parse a markdown string and return Ink-compatible React elements.
 *
 * Designed to be streaming-friendly: tolerates partial / incomplete markdown
 * (e.g. unclosed code fences) by relying on `marked`'s permissive lexer.
 */
export function renderMarkdown(text: string): React.ReactNode {
  if (!text) return null;

  const tokens = Lexer.lex(text);
  const blocks = renderBlocks(tokens);

  return <>{blocks}</>;
}
