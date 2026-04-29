import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

interface ContextEvent {
  messages: unknown[];
}

export async function recordContextSignal(
  pi: ExtensionAPI,
  event: ContextEvent,
): Promise<void> {
  const summary = JSON.stringify({ messageCount: event.messages.length });
  await pi.exec("mycelium", ["log", "context_signal", "--payload-json", summary]);
}
