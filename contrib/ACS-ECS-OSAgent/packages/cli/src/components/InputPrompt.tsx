import React, { useState, useRef } from "react";
import { Box, Text, useInput } from "ink";

import { COLORS, SYMBOLS } from "../theme.js";

export type InputPromptProps = {
  onSubmit: (text: string) => void;
  history: string[];
};

/**
 * Readline-style input prompt with multiline support, history navigation,
 * and visual cursor rendering.
 */
export function InputPrompt({ onSubmit, history }: InputPromptProps): React.ReactElement {
  const [buffer, setBuffer] = useState("");
  const [cursor, setCursor] = useState(0);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const draftRef = useRef("");

  // --- Helpers ---------------------------------------------------------------

  /** Find the start of the previous word boundary from a given position. */
  function wordBoundaryBack(text: string, pos: number): number {
    let i = pos - 1;
    // skip non-word chars
    while (i > 0 && /\s/.test(text[i]!)) i--;
    // skip word chars
    while (i > 0 && !/\s/.test(text[i - 1]!)) i--;
    return Math.max(0, i);
  }

  /** Find the end of the next word boundary from a given position. */
  function wordBoundaryForward(text: string, pos: number): number {
    let i = pos;
    // skip word chars
    while (i < text.length && !/\s/.test(text[i]!)) i++;
    // skip non-word chars
    while (i < text.length && /\s/.test(text[i]!)) i++;
    return i;
  }

  /** Get the line/col from an absolute position in the buffer. */
  function getLineCol(text: string, pos: number): { line: number; col: number } {
    const before = text.slice(0, pos);
    const lines = before.split("\n");
    return { line: lines.length - 1, col: lines[lines.length - 1]!.length };
  }

  /** Get absolute position from line/col in the buffer. */
  function getAbsPos(text: string, line: number, col: number): number {
    const lines = text.split("\n");
    let pos = 0;
    for (let i = 0; i < line && i < lines.length; i++) {
      pos += lines[i]!.length + 1; // +1 for the newline
    }
    const targetLine = lines[line] ?? "";
    return pos + Math.min(col, targetLine.length);
  }

  /** Check if cursor is on the first line of buffer. */
  function isOnFirstLine(text: string, pos: number): boolean {
    return !text.slice(0, pos).includes("\n");
  }

  /** Check if cursor is on the last line of buffer. */
  function isOnLastLine(text: string, pos: number): boolean {
    return !text.slice(pos).includes("\n");
  }

  // --- Input handler ---------------------------------------------------------

  useInput((input, key) => {
    // Submit on Enter (must come before newline check)
    if (key.return) {
      if (buffer.trim() !== "") {
        onSubmit(buffer);
      }
      setBuffer("");
      setCursor(0);
      setHistoryIndex(-1);
      draftRef.current = "";
      return;
    }

    // Ctrl+J / LF → insert newline
    // Ink delivers 0x0A as input="\n", key.ctrl=false (NOT key.ctrl && input==="j")
    if (input === "\n") {
      const newBuf = buffer.slice(0, cursor) + "\n" + buffer.slice(cursor);
      setBuffer(newBuf);
      setCursor(cursor + 1);
      return;
    }

    // --- History navigation --------------------------------------------------
    if (key.upArrow && !key.meta) {
      // Only navigate history when on the first line
      if (!isOnFirstLine(buffer, cursor)) {
        // Move cursor up within multiline buffer
        const { line, col } = getLineCol(buffer, cursor);
        if (line > 0) {
          setCursor(getAbsPos(buffer, line - 1, col));
        }
        return;
      }
      if (history.length === 0) return;
      const newIndex = historyIndex === -1
        ? history.length - 1
        : Math.max(0, historyIndex - 1);
      if (historyIndex === -1) {
        draftRef.current = buffer;
      }
      setHistoryIndex(newIndex);
      const entry = history[newIndex]!;
      setBuffer(entry);
      setCursor(entry.length);
      return;
    }

    if (key.downArrow && !key.meta) {
      // Only navigate history when on the last line
      if (!isOnLastLine(buffer, cursor)) {
        // Move cursor down within multiline buffer
        const { line, col } = getLineCol(buffer, cursor);
        const lines = buffer.split("\n");
        if (line < lines.length - 1) {
          setCursor(getAbsPos(buffer, line + 1, col));
        }
        return;
      }
      if (historyIndex === -1) return;
      const newIndex = historyIndex + 1;
      if (newIndex >= history.length) {
        setHistoryIndex(-1);
        setBuffer(draftRef.current);
        setCursor(draftRef.current.length);
      } else {
        setHistoryIndex(newIndex);
        const entry = history[newIndex]!;
        setBuffer(entry);
        setCursor(entry.length);
      }
      return;
    }

    // --- Cursor movement -----------------------------------------------------

    // Left / Ctrl+B
    if (key.leftArrow || (key.ctrl && input === "b")) {
      setCursor((c) => Math.max(0, c - 1));
      return;
    }

    // Right / Ctrl+F
    if (key.rightArrow || (key.ctrl && input === "f")) {
      setCursor((c) => Math.min(buffer.length, c + 1));
      return;
    }

    // Ctrl+A / Home → beginning
    if (key.ctrl && input === "a") {
      setCursor(0);
      return;
    }

    // Ctrl+E / End → end
    if (key.ctrl && input === "e") {
      setCursor(buffer.length);
      return;
    }

    // Alt+F → forward one word
    if (key.meta && input === "f") {
      setCursor(wordBoundaryForward(buffer, cursor));
      return;
    }

    // Alt+B → backward one word
    if (key.meta && input === "b") {
      setCursor(wordBoundaryBack(buffer, cursor));
      return;
    }

    // --- Deletion ------------------------------------------------------------

    // Ctrl+D → delete at cursor
    if (key.ctrl && input === "d") {
      if (buffer.length === 0) return;
      const newBuf = buffer.slice(0, cursor) + buffer.slice(cursor + 1);
      setBuffer(newBuf);
      return;
    }

    // Ctrl+K → kill from cursor to end of line
    if (key.ctrl && input === "k") {
      const nlIndex = buffer.indexOf("\n", cursor);
      const end = nlIndex === -1 ? buffer.length : nlIndex;
      setBuffer(buffer.slice(0, cursor) + buffer.slice(end));
      return;
    }

    // Ctrl+U → kill from start of line to cursor
    if (key.ctrl && input === "u") {
      const lastNl = buffer.lastIndexOf("\n", cursor - 1);
      const lineStart = lastNl + 1;
      setBuffer(buffer.slice(0, lineStart) + buffer.slice(cursor));
      setCursor(lineStart);
      return;
    }

    // Ctrl+W → kill previous word
    if (key.ctrl && input === "w") {
      const boundary = wordBoundaryBack(buffer, cursor);
      setBuffer(buffer.slice(0, boundary) + buffer.slice(cursor));
      setCursor(boundary);
      return;
    }

    // Backspace / Delete key (macOS ⌫ sends 0x7F → Ink maps to key.delete)
    if (key.backspace || key.delete) {
      if (cursor === 0) return;
      setBuffer(buffer.slice(0, cursor - 1) + buffer.slice(cursor));
      setCursor(cursor - 1);
      return;
    }

    // --- Ignore other control/meta sequences ---------------------------------
    if (key.ctrl || key.meta) {
      return;
    }

    // --- Normal character insertion -------------------------------------------
    if (input && input.charCodeAt(0) >= 0x20) {
      const newBuf = buffer.slice(0, cursor) + input + buffer.slice(cursor);
      setBuffer(newBuf);
      setCursor(cursor + input.length);
    }
  });

  // --- Rendering -------------------------------------------------------------

  const lines = buffer.split("\n");

  return (
    <Box flexDirection="column">
      {lines.map((line, lineIdx) => {
        const lineStart = lines.slice(0, lineIdx).reduce(
          (acc, l) => acc + l.length + 1,
          0,
        );
        const cursorInLine = cursor - lineStart;
        const isCursorOnThisLine = cursor >= lineStart && cursor <= lineStart + line.length;
        const prefix = lineIdx === 0
          ? <Text color={COLORS.prompt} bold>{SYMBOLS.promptPrefix}</Text>
          : <Text color={COLORS.border}>{"\u2026 "}</Text>;

        return (
          <Text key={lineIdx}>
            {prefix}
            {isCursorOnThisLine ? (
              <>
                <Text color={COLORS.userInput}>{line.slice(0, cursorInLine)}</Text>
                <Text inverse color={COLORS.userInput}>
                  {cursorInLine < line.length ? line[cursorInLine] : " "}
                </Text>
                <Text color={COLORS.userInput}>{line.slice(cursorInLine + 1)}</Text>
              </>
            ) : (
              <Text color={COLORS.userInput}>{line}</Text>
            )}
          </Text>
        );
      })}
    </Box>
  );
}
