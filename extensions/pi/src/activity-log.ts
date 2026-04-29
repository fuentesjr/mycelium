import type { ContextEvent, ExtensionAPI } from "@mariozechner/pi-coding-agent";

export async function recordContextSignal(
  pi: ExtensionAPI,
  event: ContextEvent,
): Promise<void> {
  const messages = event.messages;
  const last = messages[messages.length - 1];
  const payload: Record<string, unknown> = { messageCount: messages.length };
  if (last && "role" in last && typeof last.role === "string") {
    payload.lastRole = last.role;
  }
  await pi.exec("mycelium", [
    "log",
    "context_signal",
    "--payload-json",
    JSON.stringify(payload),
  ]);
}
