import React, { useState, useEffect, useCallback } from "react";
import { Box, Static, Text, useApp, useInput } from "ink";

import type { AppServer } from "core";

import { COLORS, SYMBOLS } from "./theme.js";
import { InputPrompt } from "./components/InputPrompt.js";
import { MessageStream } from "./components/MessageStream.js";
import { ToolCall } from "./components/ToolCall.js";
import { CommandExecution } from "./components/CommandExecution.js";
import { SkillBadge } from "./components/SkillBadge.js";
import { Spinner } from "./components/Spinner.js";
import { ErrorPanel } from "./components/ErrorPanel.js";
import { StatusBar } from "./components/StatusBar.js";
import {
  useTurn,
  type ApprovalDecision,
  type StaticItem,
  type ToolCallState,
  type TurnSegment,
} from "./hooks/useTurn.js";

export type AppProps = {
  server: AppServer;
  threadId: string;
  approvalMode: "prompt" | "yolo";
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Try to extract the skill name from activate_skill JSON arguments. */
function parseSkillName(args?: string): string {
  if (!args) return "unknown";
  try {
    const parsed = JSON.parse(args) as { name?: string };
    return parsed.name ?? "unknown";
  } catch {
    return "unknown";
  }
}

/** Map ToolCallStatus to SkillBadge status. */
function toSkillStatus(status: import("./hooks/useTurn.js").ToolCallStatus): "activating" | "active" | "error" {
  switch (status) {
    case "running":
    case "pending_approval":
      return "activating";
    case "completed":
      return "active";
    case "failed":
    case "denied":
      return "error";
  }
}

/** Render a single tool call entry, dispatching to the right component. */
function renderToolCall(
  tc: ToolCallState,
  resolveApproval?: (decision: ApprovalDecision) => void,
): React.ReactElement {
  if (tc.itemType === "commandExecution") {
    return (
      <CommandExecution
        key={tc.id}
        state={tc}
        onDecision={resolveApproval}
      />
    );
  }
  // Skill activation (activate_skill tool call)
  if (tc.itemType === "toolCall" && tc.toolName === "activate_skill") {
    return (
      <SkillBadge
        key={tc.id}
        name={parseSkillName(tc.arguments)}
        status={toSkillStatus(tc.status)}
      />
    );
  }
  // MCP or generic tool call
  return (
    <ToolCall
      key={tc.id}
      toolName={tc.toolName}
      status={tc.status}
      itemType={tc.itemType}
      arguments={tc.arguments}
      output={tc.output}
    />
  );
}

/** Render a single turn segment, preserving chronological order. */
function renderSegment(
  segment: TurnSegment,
  index: number,
  resolveApproval?: (decision: ApprovalDecision) => void,
): React.ReactElement {
  if (segment.type === "text") {
    return (
      <Box key={`text-${index}`} paddingLeft={2}>
        <MessageStream text={segment.content} />
      </Box>
    );
  }
  return renderToolCall(segment.toolCall, resolveApproval);
}

// ---------------------------------------------------------------------------
// App
// ---------------------------------------------------------------------------

export function App({ server, threadId, approvalMode }: AppProps): React.ReactElement {
  const { exit } = useApp();
  const { state, startTurn, resolveApproval } = useTurn(server, threadId, approvalMode);
  const [welcomeShown] = useState(`Thread ${threadId} started. Type "/quit" to exit.`);
  const [exiting, setExiting] = useState(false);
  const [history, setHistory] = useState<string[]>([]);

  // Handle /quit and /exit commands
  const handleSubmit = useCallback((text: string) => {
    const trimmed = text.trim();
    if (trimmed === "/quit" || trimmed === "/exit" || trimmed === "exit") {
      setExiting(true);
      exit();
      return;
    }
    setHistory((prev) => [...prev, trimmed]);
    startTurn(trimmed);
  }, [exit, startTurn]);

  // Ctrl+C graceful exit
  useInput((_input, key) => {
    if (key.ctrl && _input === "c") {
      setExiting(true);
      exit();
    }
  });

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      // Generator cleanup happens via useTurn's own teardown
    };
  }, []);

  return (
    <Box flexDirection="column">
      {/* Static area: append-only frozen blocks. The user prompt is frozen on
          turn start; each segment (text block or tool call) is frozen as soon
          as it is finalized — that is, when a later segment starts after it,
          when a tool call reaches a terminal status, or at turn end. The
          array reference grows but never mutates earlier entries, so Ink
          emits each block exactly once and never re-renders it on subsequent
          state updates. This progressive promotion is what keeps both the
          user prompt and completed tool-call boxes (e.g. SkillBadge /
          CommandExecution borders) from ghosting on every streaming delta. */}
      <Static items={state.staticItems}>
        {(item: StaticItem) => {
          if (item.kind === "prompt") {
            const promptLines = item.userInput.split("\n");
            return (
              <Box key={item.key} marginBottom={1} flexDirection="column">
                {promptLines.map((pLine, pIdx) => (
                  <Text key={pIdx}>
                    {pIdx === 0
                      ? <Text color={COLORS.prompt} bold>{SYMBOLS.promptPrefix}</Text>
                      : <Text color={COLORS.border}>{"\u2026 "}</Text>}
                    <Text color={COLORS.userInput}>{pLine}</Text>
                  </Text>
                ))}
              </Box>
            );
          }
          if (item.kind === "segment") {
            return (
              <Box key={item.key} flexDirection="column">
                {renderSegment(item.segment, 0)}
              </Box>
            );
          }
          // turnEnd: a one-line vertical gap between turns.
          return <Text key={item.key}>{" "}</Text>;
        }}
      </Static>

      {/* Welcome message */}
      {state.staticItems.length === 0 && state.phase === "idle" && (
        <Text color={COLORS.border}>{welcomeShown}</Text>
      )}

      {/* Live area: only the still-active tail of the current turn. Segments
          with index < frozenSegmentCount have already been promoted into
          <Static> above and must not be re-rendered here, otherwise xterm.js
          would ghost their multi-line content on every state update. */}
      {state.phase !== "idle" && state.segments.length > state.frozenSegmentCount && (
        <Box flexDirection="column">
          {state.segments.slice(state.frozenSegmentCount).map((seg, i) =>
            renderSegment(seg, state.frozenSegmentCount + i, resolveApproval),
          )}
        </Box>
      )}

      {state.phase === "thinking" && <Spinner label="Thinking..." />}

      {/* Error display */}
      {state.error && (
        <ErrorPanel
          message={state.error}
          hint="Check your API key, network connectivity, or try again."
        />
      )}

      {/* Goodbye message on exit */}
      {exiting && (
        <Text dimColor>Goodbye!</Text>
      )}

      {/* Input prompt when idle */}
      {state.phase === "idle" && (
        <Box marginTop={1} flexDirection="column">
          <InputPrompt onSubmit={handleSubmit} history={history} />
          <StatusBar />
        </Box>
      )}
    </Box>
  );
}
