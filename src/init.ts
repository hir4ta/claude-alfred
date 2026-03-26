import { defineCommand } from "citty";

export const initCommand = defineCommand({
	meta: { description: "Set up alfred hooks, skills, agents, and rules in ~/.claude/" },
	args: {
		force: { type: "boolean", description: "Overwrite existing configuration", default: false },
	},
	async run({ args: _args }) {
		// TODO: Phase 1 — write hooks to settings.json, create .alfred/gates.json
		console.log("alfred init: not yet implemented");
	},
});
