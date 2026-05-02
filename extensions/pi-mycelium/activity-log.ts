import type {
  ContextEvent,
  ExtensionAPI,
  SessionStartEvent,
} from "@mariozechner/pi-coding-agent";

export async function recordContextSignal(
  pi: ExtensionAPI,
  binaryPath: string,
  event: ContextEvent,
): Promise<void> {
  const messages = event.messages;
  const last = messages[messages.length - 1];
  const payload: Record<string, unknown> = { messageCount: messages.length };
  if (last && "role" in last && typeof last.role === "string") {
    payload.lastRole = last.role;
  }
  await pi.exec(binaryPath, [
    "log",
    "context_signal",
    "--payload-json",
    JSON.stringify(payload),
  ]);
}

const BOUNDARY_REASONS: ReadonlySet<SessionStartEvent["reason"]> = new Set([
  "new",
  "resume",
  "fork",
]);

export async function recordSessionBoundary(
  pi: ExtensionAPI,
  binaryPath: string,
  reason: SessionStartEvent["reason"],
): Promise<void> {
  if (!BOUNDARY_REASONS.has(reason)) return;
  await pi.exec(binaryPath, ["log", `session_${reason}`]);
}
