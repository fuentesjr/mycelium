import { createRequire } from "node:module";
import type {
	ExtensionAPI,
	SessionCompactEvent,
	SessionShutdownEvent,
	SessionStartEvent,
} from "@earendil-works/pi-coding-agent";

const require = createRequire(import.meta.url);

const DEFAULT_HARNESS = "pi.dev";
const DEFAULT_ADAPTER_VERSION = resolveAdapterVersion();

export interface ActivityRecorderOptions {
	harness?: string;
	adapterVersion?: string;
}

type JsonObject = Record<string, unknown>;

export class ActivityLogRecorder {
	private readonly harness: string;
	private readonly adapterVersion: string;
	private seq = 0;

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
