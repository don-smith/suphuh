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

type AgentState = "working" | "blocked" | "idle";

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

  let blockedCount = 0;
  let blockedMessage: string | undefined;
  let agentActive = false;
  let idleTimer: ReturnType<typeof setTimeout> | undefined;

  function setIdleSoon() {
    if (idleTimer) clearTimeout(idleTimer);
    idleTimer = setTimeout(() => {
      idleTimer = undefined;
      if (!agentActive && blockedCount === 0) void publish("idle");
    }, 250);
    idleTimer.unref?.();
  }

  function publishDesired() {
    if (blockedCount > 0) {
      void publish("blocked", blockedMessage);
    } else if (agentActive) {
      void publish("working");
    } else {
      setIdleSoon();
    }
  }

  pi.events.on("herdr:blocked", (data: any) => {
    if (!data?.active) {
      blockedCount = Math.max(0, blockedCount - 1);
      if (blockedCount === 0) blockedMessage = undefined;
      publishDesired();
      return;
    }

    blockedCount += 1;
    blockedMessage = data.label;
    publishDesired();
  });

  pi.on("session_start", () => {
    void publish("idle");
  });

  pi.on("agent_start", () => {
    if (idleTimer) clearTimeout(idleTimer);
    idleTimer = undefined;
    agentActive = true;
    publishDesired();
  });

  pi.on("agent_end", () => {
    agentActive = false;
    publishDesired();
  });

  pi.on("session_shutdown", () => {
    if (idleTimer) clearTimeout(idleTimer);
    void clear();
  });
}
