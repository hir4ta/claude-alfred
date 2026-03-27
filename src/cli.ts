declare const __ALFRED_VERSION__: string;

import { defineCommand, runMain } from "citty";
import { dispatch } from "./hooks/dispatcher.ts";

const main = defineCommand({
	meta: {
		name: "alfred",
		version: typeof __ALFRED_VERSION__ !== "undefined" ? __ALFRED_VERSION__ : "dev",
		description: "Claude Code quality butler",
	},
	subCommands: {
		init: () => import("./init.ts").then((m) => m.initCommand),
		doctor: () => import("./doctor.ts").then((m) => m.doctorCommand),
		reset: () => import("./reset.ts").then((m) => m.resetCommand),
		status: () => import("./status.ts").then((m) => m.statusCommand),
		hook: defineCommand({
			meta: { description: "Run a hook handler" },
			args: {
				event: { type: "positional", description: "Hook event name", required: true },
			},
			async run({ args }) {
				await dispatch(args.event);
			},
		}),
	},
});

runMain(main);
