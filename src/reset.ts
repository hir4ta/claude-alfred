import { existsSync, readdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { defineCommand } from "citty";

const KEEP_ON_HISTORY = ["gate-history.json", "metrics.json", "session-outcomes.json"];

export function runReset(keepHistory: boolean): { deleted: number; kept: number } {
	const stateDir = join(process.cwd(), ".alfred", ".state");
	if (!existsSync(stateDir)) return { deleted: 0, kept: 0 };

	const files = readdirSync(stateDir).filter((f) => f.endsWith(".json"));
	let deleted = 0;
	let kept = 0;

	for (const file of files) {
		if (keepHistory && KEEP_ON_HISTORY.includes(file)) {
			kept++;
			continue;
		}
		rmSync(join(stateDir, file), { force: true });
		deleted++;
	}

	return { deleted, kept };
}

export const resetCommand = defineCommand({
	meta: { description: "Reset alfred state" },
	args: {
		keepHistory: {
			type: "boolean",
			alias: "keep-history",
			description: "Keep gate-history and metrics",
			default: false,
		},
	},
	async run({ args }) {
		const result = runReset(args.keepHistory);
		console.log(`Reset: ${result.deleted} file(s) deleted, ${result.kept} kept`);
	},
});
