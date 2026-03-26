import { defineCommand } from "citty";

export type CheckStatus = "ok" | "fail" | "warn";
export interface CheckResult {
	name: string;
	status: CheckStatus;
	message: string;
}

/** Run all health checks and return results */
export function runChecks(): CheckResult[] {
	// TODO: implement in Phase 3
	return [];
}

export const doctorCommand = defineCommand({
	meta: { description: "Check alfred health" },
	async run() {
		const results = runChecks();
		for (const r of results) {
			const tag = r.status === "ok" ? "[OK]" : r.status === "fail" ? "[FAIL]" : "[WARN]";
			console.log(`${tag} ${r.message}`);
		}
		const hasFail = results.some((r) => r.status === "fail");
		if (hasFail) process.exit(1);
	},
});
