/**
 * Minimal hook entry point for plugin distribution.
 * Called as: node dist/hook.mjs <event>
 *
 * Replaces the citty CLI for hook dispatch only.
 */
import { dispatch } from "./hooks/dispatcher.ts";

const event = process.argv[2];
if (!event) {
	process.stderr.write("Usage: hook.mjs <event>\n");
	process.exit(1);
}

dispatch(event).catch((err) => {
	// fail-open: don't let qult crash Claude Code
	if (err instanceof Error) {
		process.stderr.write(`[qult] ${err.message}\n`);
	}
});
