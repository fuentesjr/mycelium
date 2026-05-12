import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import type {
	ContextEvent,
	ExtensionAPI,
	ExtensionEvent,
	SessionCompactEvent,
	SessionShutdownEvent,
	SessionStartEvent,
	TurnEndEvent,
	TurnStartEvent,
} from "@earendil-works/pi-coding-agent";

export type ToolExecutionStartEvent = Extract<
	ExtensionEvent,
	{ type: "tool_execution_start" }
>;
export type ToolExecutionEndEvent = Extract<
	ExtensionEvent,
	{ type: "tool_execution_end" }
>;

const require = createRequire(import.meta.url);

const DEFAULT_HARNESS = "pi.dev";
const DEFAULT_ADAPTER_VERSION = resolveAdapterVersion();

export interface ActivityRecorderOptions {
	harness?: string;
	adapterVersion?: string;
}

export interface ContextCheckpointOptions {
	turnIndex?: number;
}

type JsonObject = Record<string, unknown>;

interface CheckpointPayloadResult {
	fingerprint: string;
	payload: JsonObject;
	messageCount: number;
}

export class ActivityLogRecorder {
	private readonly harness: string;
	private readonly adapterVersion: string;
	private seq = 0;
	private lastCheckpointFingerprint: string | null = null;
	private lastCheckpointMessageCount: number | null = null;
	private suppressedCheckpointDuplicates = 0;
	private readonly toolStartedAt = new Map<string, number>();

	constructor(options: ActivityRecorderOptions = {}) {
		this.harness = options.harness ?? DEFAULT_HARNESS;
		this.adapterVersion = options.adapterVersion ?? DEFAULT_ADAPTER_VERSION;
	}

	async recordSessionBoundary(
		pi: ExtensionAPI,
		binaryPath: string,
		reason: SessionStartEvent["reason"],
	): Promise<void> {
		await this.log(pi, binaryPath, `session_${reason}`, {
			session_reason: reason,
		});
	}

	async recordSessionShutdown(
		pi: ExtensionAPI,
		binaryPath: string,
		event: SessionShutdownEvent,
	): Promise<void> {
		await this.log(pi, binaryPath, "session_shutdown", {
			session_reason: event.reason,
			...(event.targetSessionFile
				? { target_session_file: event.targetSessionFile }
				: {}),
		});
	}

	async recordContextCheckpoint(
		pi: ExtensionAPI,
		binaryPath: string,
		event: ContextEvent,
		options: ContextCheckpointOptions = {},
	): Promise<void> {
		const checkpoint = buildContextCheckpointPayload(event, options);
		if (checkpoint.fingerprint === this.lastCheckpointFingerprint) {
			this.suppressedCheckpointDuplicates += 1;
			return;
		}

		const payload = { ...checkpoint.payload };
		if (this.lastCheckpointMessageCount !== null) {
			payload.message_delta =
				checkpoint.messageCount - this.lastCheckpointMessageCount;
		}
		if (this.suppressedCheckpointDuplicates > 0) {
			payload.suppressed_duplicates = this.suppressedCheckpointDuplicates;
		}

		this.lastCheckpointFingerprint = checkpoint.fingerprint;
		this.lastCheckpointMessageCount = checkpoint.messageCount;
		this.suppressedCheckpointDuplicates = 0;

		await this.log(pi, binaryPath, "context_checkpoint", payload);
	}

	async recordTurnStart(
		pi: ExtensionAPI,
		binaryPath: string,
		event: TurnStartEvent,
	): Promise<void> {
		await this.log(pi, binaryPath, "turn_start", {
			turn_index: event.turnIndex,
			timestamp_ms: event.timestamp,
		});
	}

	async recordTurnEnd(
		pi: ExtensionAPI,
		binaryPath: string,
		event: TurnEndEvent,
	): Promise<void> {
		await this.log(pi, binaryPath, "turn_end", {
			turn_index: event.turnIndex,
			last_role: roleOf(event.message),
			tool_result_count: event.toolResults.length,
			tool_names: event.toolResults.map((r) => r.toolName),
			...assistantPayloadFields(event.message),
		});
	}

	async recordToolStart(
		pi: ExtensionAPI,
		binaryPath: string,
		event: ToolExecutionStartEvent,
	): Promise<void> {
		await this.log(pi, binaryPath, "tool_start", {
			tool_call_id: event.toolCallId,
			tool_name: event.toolName,
		});
		this.toolStartedAt.set(event.toolCallId, Date.now());
	}

	async recordToolEnd(
		pi: ExtensionAPI,
		binaryPath: string,
		event: ToolExecutionEndEvent,
	): Promise<void> {
		const startedAt = this.toolStartedAt.get(event.toolCallId);
		this.toolStartedAt.delete(event.toolCallId);

		await this.log(pi, binaryPath, "tool_end", {
			tool_call_id: event.toolCallId,
			tool_name: event.toolName,
			is_error: event.isError,
			...(startedAt !== undefined
				? { duration_ms: Math.max(0, Date.now() - startedAt) }
				: {}),
			...toolResultPayloadFields(event.result),
		});
	}

	async recordCompaction(
		pi: ExtensionAPI,
		binaryPath: string,
		event: SessionCompactEvent,
	): Promise<void> {
		await this.log(pi, binaryPath, "compaction", {
			from_extension: event.fromExtension,
		});
	}

	private async log(
		pi: ExtensionAPI,
		binaryPath: string,
		op: string,
		payload: JsonObject = {},
	): Promise<void> {
		const enriched = {
			...payload,
			harness: this.harness,
			adapter_version: this.adapterVersion,
			seq: ++this.seq,
		};
		await pi.exec(binaryPath, [
			"log",
			op,
			"--payload-json",
			JSON.stringify(enriched),
		]);
	}
}

export function createActivityLogRecorder(
	options?: ActivityRecorderOptions,
): ActivityLogRecorder {
	return new ActivityLogRecorder(options);
}

function resolveAdapterVersion(): string {
	try {
		const pkg = require("./package.json") as { version?: unknown };
		return typeof pkg.version === "string" ? pkg.version : "unknown";
	} catch {
		return "unknown";
	}
}

export async function recordContextCheckpoint(
	pi: ExtensionAPI,
	binaryPath: string,
	event: ContextEvent,
): Promise<void> {
	await createActivityLogRecorder().recordContextCheckpoint(
		pi,
		binaryPath,
		event,
	);
}

/**
 * Legacy helper kept for older imports/tests. New adapters should emit
 * `context_checkpoint`; old `context_signal` entries remain readable in logs.
 */
export async function recordContextSignal(
	pi: ExtensionAPI,
	binaryPath: string,
	event: ContextEvent,
): Promise<void> {
	const messages = event.messages;
	const payload: JsonObject = { messageCount: messages.length };
	const lastRole = roleOf(messages[messages.length - 1]);
	if (lastRole) payload.lastRole = lastRole;
	await pi.exec(binaryPath, [
		"log",
		"context_signal",
		"--payload-json",
		JSON.stringify(payload),
	]);
}

export async function recordSessionBoundary(
	pi: ExtensionAPI,
	binaryPath: string,
	reason: SessionStartEvent["reason"],
): Promise<void> {
	await createActivityLogRecorder().recordSessionBoundary(
		pi,
		binaryPath,
		reason,
	);
}

function buildContextCheckpointPayload(
	event: ContextEvent,
	options: ContextCheckpointOptions,
): CheckpointPayloadResult {
	const messages = event.messages;
	const fingerprint = fingerprintContext(messages);
	const payload: JsonObject = {
		message_count: messages.length,
		role_counts: countRoles(messages),
		fingerprint,
		...latestAssistantPayloadFields(messages),
	};

	if (options.turnIndex !== undefined) payload.turn_index = options.turnIndex;

	const lastRole = roleOf(messages[messages.length - 1]);
	if (lastRole) payload.last_role = lastRole;

	return { fingerprint, payload, messageCount: messages.length };
}

function countRoles(
	messages: ContextEvent["messages"],
): Record<string, number> {
	const counts: Record<string, number> = {};
	for (const message of messages) {
		const role = roleOf(message) ?? "unknown";
		counts[role] = (counts[role] ?? 0) + 1;
	}
	return counts;
}

function fingerprintContext(messages: ContextEvent["messages"]): string {
	const stable = messages.map((message) =>
		summarizeMessageForFingerprint(message),
	);
	return sha256JSON({ messages: stable });
}

function summarizeMessageForFingerprint(message: unknown): unknown {
	if (!isRecord(message)) return { kind: typeof message };
	const role = typeof message.role === "string" ? message.role : undefined;
	const timestamp = scalar(message.timestamp);

	if (role === "user") {
		return {
			role,
			timestamp,
			content_shape: summarizeContentShape(message.content),
		};
	}

	if (role === "assistant") {
		return {
			role,
			timestamp,
			provider: scalar(message.provider),
			model: scalar(message.model),
			response_id: scalar(message.responseId),
			stop_reason: scalar(message.stopReason),
			content_shape: summarizeContentShape(message.content),
		};
	}

	if (role === "toolResult") {
		return {
			role,
			timestamp,
			tool_call_id: scalar(message.toolCallId),
			tool_name: scalar(message.toolName),
			is_error: scalar(message.isError),
			content_shape: summarizeContentShape(message.content),
		};
	}

	return {
		role,
		custom_type: scalar(message.customType),
		timestamp,
		keys: Object.keys(message).sort(),
	};
}

function summarizeContentShape(content: unknown): unknown {
	if (typeof content === "string") {
		return { kind: "text", chars: content.length };
	}
	if (!Array.isArray(content)) {
		return { kind: typeof content };
	}
	return content.map((item) => {
		if (!isRecord(item)) return { kind: typeof item };
		if (item.type === "text") {
			return {
				type: "text",
				chars: typeof item.text === "string" ? item.text.length : 0,
			};
		}
		if (item.type === "thinking") {
			return {
				type: "thinking",
				chars: typeof item.thinking === "string" ? item.thinking.length : 0,
				redacted: item.redacted === true,
			};
		}
		if (item.type === "image") {
			return {
				type: "image",
				mime_type: scalar(item.mimeType),
				data_chars: typeof item.data === "string" ? item.data.length : 0,
			};
		}
		if (item.type === "toolCall") {
			const args = isRecord(item.arguments)
				? Object.keys(item.arguments).sort()
				: [];
			return {
				type: "toolCall",
				id: scalar(item.id),
				name: scalar(item.name),
				arg_keys: args,
			};
		}
		return { type: scalar(item.type), keys: Object.keys(item).sort() };
	});
}

function latestAssistantPayloadFields(
	messages: ContextEvent["messages"],
): JsonObject {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const fields = assistantPayloadFields(messages[i]);
		if (Object.keys(fields).length > 0) return fields;
	}
	return {};
}

function assistantPayloadFields(message: unknown): JsonObject {
	if (!isRecord(message) || message.role !== "assistant") return {};
	return {
		...(typeof message.provider === "string"
			? { provider: message.provider }
			: {}),
		...(typeof message.model === "string" ? { model: message.model } : {}),
		...(typeof message.stopReason === "string"
			? { stop_reason: message.stopReason }
			: {}),
		...usagePayloadFields(message.usage),
	};
}

function usagePayloadFields(usage: unknown): JsonObject {
	if (!isRecord(usage)) return {};
	const result: JsonObject = {};
	const input = numberValue(usage.input);
	const output = numberValue(usage.output);
	const cacheRead = numberValue(usage.cacheRead);
	const cacheWrite = numberValue(usage.cacheWrite);
	const totalTokens = numberValue(usage.totalTokens);
	if (
		input !== undefined ||
		output !== undefined ||
		cacheRead !== undefined ||
		cacheWrite !== undefined ||
		totalTokens !== undefined
	) {
		result.usage = {
			...(input !== undefined ? { input } : {}),
			...(output !== undefined ? { output } : {}),
			...(cacheRead !== undefined ? { cache_read: cacheRead } : {}),
			...(cacheWrite !== undefined ? { cache_write: cacheWrite } : {}),
			...(totalTokens !== undefined ? { total_tokens: totalTokens } : {}),
		};
	}

	const cost = isRecord(usage.cost) ? numberValue(usage.cost.total) : undefined;
	if (cost !== undefined) result.cost = { total: cost };
	return result;
}

function toolResultPayloadFields(result: unknown): JsonObject {
	if (!isRecord(result)) return {};
	return {
		output_chars: outputChars(result.content),
		...exitCodePayload(result.details),
	};
}

function outputChars(content: unknown): number {
	if (!Array.isArray(content)) return 0;
	let total = 0;
	for (const item of content) {
		if (
			isRecord(item) &&
			item.type === "text" &&
			typeof item.text === "string"
		) {
			total += item.text.length;
		}
	}
	return total;
}

function exitCodePayload(details: unknown): JsonObject {
	if (!isRecord(details)) return {};
	const exitCode =
		numberValue(details.exit_code) ?? numberValue(details.exitCode);
	return exitCode === undefined ? {} : { exit_code: exitCode };
}

function roleOf(message: unknown): string | undefined {
	return isRecord(message) && typeof message.role === "string"
		? message.role
		: undefined;
}

function sha256JSON(value: unknown): string {
	return `sha256:${createHash("sha256").update(JSON.stringify(value)).digest("hex")}`;
}

function scalar(value: unknown): string | number | boolean | undefined {
	switch (typeof value) {
		case "string":
		case "number":
		case "boolean":
			return value;
		default:
			return undefined;
	}
}

function numberValue(value: unknown): number | undefined {
	return typeof value === "number" && Number.isFinite(value)
		? value
		: undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null;
}
