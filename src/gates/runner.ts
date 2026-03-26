import type { GateDefinition } from "../types.ts";

export interface GateResult {
	name: string;
	passed: boolean;
	output: string;
}

/** Run a single gate command. Returns pass/fail + output. */
export function runGate(name: string, gate: GateDefinition, file?: string): GateResult {
	const command = file ? gate.command.replace("{file}", file) : gate.command;
	const timeout = gate.timeout ?? 10_000;

	let result: ReturnType<typeof Bun.spawnSync>;
	try {
		result = Bun.spawnSync(["sh", "-c", command], {
			cwd: process.cwd(),
			timeout,
			stdio: ["ignore", "pipe", "pipe"],
			env: {
				...process.env,
				PATH: `${process.cwd()}/node_modules/.bin:${process.env.PATH}`,
			},
		});
	} catch {
		return { name, passed: false, output: `Gate "${name}" failed to execute` };
	}

	const stdout = result.stdout?.toString() ?? "";
	const stderr = result.stderr?.toString() ?? "";
	const output = (stdout + stderr).slice(0, 1000);

	return {
		name,
		passed: result.exitCode === 0,
		output: output || (result.exitCode !== 0 ? `Exit code ${result.exitCode}` : ""),
	};
}
