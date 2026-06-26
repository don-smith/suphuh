/**
 * suphuh status extension for Pi.
 *
 * Writes lightweight pane status reports to ~/.suphuh/status/<pane>.json.
 * suphuh reads these files to show agent status without needing to control
 * the running agent's terminal stdin/stdout.
 */

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { mkdir, rename, rm, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join } from "node:path";

const paneId = process.env.TMUX_PANE;
const statusDir = process.env.SUPHUH_STATUS_DIR || join(homedir(), ".suphuh", "status");

type AgentState = "working" | "waiting" | "idle";

function enabled(): boolean {
  return !!paneId;
}

function paneFileName(id: string): string {
  return id
    .replaceAll("%", "pct_")
    .replaceAll("$", "session_")
    .replaceAll("/", "_")
    .replaceAll("\\", "_")
    .replaceAll(":", "_");
}

function statusPath(): string {
  return join(statusDir, `${paneFileName(paneId!)}.json`);
}

async function publish(state: AgentState, message?: string): Promise<void> {
  if (!enabled()) return;

  const path = statusPath();
  const tmp = `${path}.${process.pid}.${Date.now()}.${Math.random().toString(36).slice(2)}.tmp`;
  const report = {
    pane_id: paneId,
    agent: "pi",
    state,
    message,
    updated_at: new Date().toISOString(),
  };

  await mkdir(dirname(path), { recursive: true });
  await writeFile(tmp, `${JSON.stringify(report)}\n`, "utf8");
  await rename(tmp, path);
}

async function clear(): Promise<void> {
  if (!enabled()) return;
  await rm(statusPath(), { force: true });
}

export default function (pi: ExtensionAPI) {
  if (!enabled()) return;

  let toolCount = 0;
  let waitingTimer: ReturnType<typeof setTimeout> | undefined;
  let agentActive = false;
  let idleTimer: ReturnType<typeof setTimeout> | undefined;

  function setIdleSoon() {
    if (idleTimer) clearTimeout(idleTimer);
    idleTimer = setTimeout(() => {
      idleTimer = undefined;
      if (!agentActive && toolCount === 0) void publish("idle");
    }, 250);
    idleTimer.unref?.();
  }

  function maybePublishWaiting() {
    if (toolCount > 0 && agentActive) {
      void publish("waiting");
    }
  }

  function publishDesired() {
    if (toolCount > 0) {
      // Don't immediately publish waiting — debounce to skip fast tools.
      return;
    }
    if (agentActive) {
      void publish("working");
    } else {
      setIdleSoon();
    }
  }

  pi.on("tool_execution_start", () => {
    toolCount++;
    // If a tool takes longer than 200ms, assume it's blocking for user input.
    if (!waitingTimer) {
      waitingTimer = setTimeout(() => {
        waitingTimer = undefined;
        maybePublishWaiting();
      }, 200);
      waitingTimer.unref?.();
    }
  });

  pi.on("tool_execution_end", () => {
    toolCount = Math.max(0, toolCount - 1);
    if (toolCount === 0) {
      if (waitingTimer) {
        clearTimeout(waitingTimer);
        waitingTimer = undefined;
      }
      publishDesired();
    }
  });

  pi.on("session_start", () => {
    void publish("idle");
  });

  pi.on("agent_start", () => {
    if (idleTimer) clearTimeout(idleTimer);
    idleTimer = undefined;
    agentActive = true;
    toolCount = 0;
    publishDesired();
  });

  pi.on("agent_end", () => {
    agentActive = false;
    toolCount = 0;
    if (waitingTimer) {
      clearTimeout(waitingTimer);
      waitingTimer = undefined;
    }
    publishDesired();
  });

  pi.on("session_shutdown", () => {
    if (idleTimer) clearTimeout(idleTimer);
    if (waitingTimer) clearTimeout(waitingTimer);
    void clear();
  });
}
