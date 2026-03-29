import { flushAll } from "../state/flush.ts";

/** Current hook event name, set by dispatcher before calling handler */
let _currentEvent = "unknown";
export function setCurrentEvent(event: string): void {
	_currentEvent = event;
}
export function getCurrentEvent(): string {
	return _currentEvent;
}

/** DENY: block the action (exit 2).
 * stderr-only, no stdout — bypasses plugin hook output bug (#16538).
 * Only valid for: PreToolUse */
export function deny(reason: string): never {
	try {
		flushAll();
	} catch {
		/* fail-open */
	}
	process.stderr.write(reason);
	process.exit(2);
}

/** Block Claude from stopping (exit 2).
 * stderr-only, no stdout — bypasses plugin hook output bug (#16538).
 * Valid for: Stop, SubagentStop */
export function block(reason: string): never {
	try {
		flushAll();
	} catch {
		/* fail-open */
	}
	process.stderr.write(reason);
	process.exit(2);
}
