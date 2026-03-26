import { defineCommand } from "citty";

export const doctorCommand = defineCommand({
	meta: { description: "Check alfred health" },
	async run() {
		// TODO: Phase 1 — check hooks registered, gates.json exists
		console.log("alfred doctor: not yet implemented");
	},
});
