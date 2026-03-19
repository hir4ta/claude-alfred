import { describe, expect, it } from "vitest";
import { emitAdditionalContext, extractSection, notifyUser } from "../dispatcher.js";

describe("extractSection", () => {
	it("extracts section content between headings", () => {
		const md = "## Intro\nHello\n## Next Steps\n- [ ] Todo\n- [x] Done\n## Other\nStuff";
		expect(extractSection(md, "## Next Steps")).toBe("- [ ] Todo\n- [x] Done");
	});

	it("extracts section at end of document", () => {
		const md = "## First\nA\n## Last\nFinal content";
		expect(extractSection(md, "## Last")).toBe("Final content");
	});

	it("returns empty string when section not found", () => {
		const md = "## Intro\nHello";
		expect(extractSection(md, "## Missing")).toBe("");
	});

	it("handles heading with extra text after match", () => {
		const md = "## Next Steps (updated)\nContent here\n## Other\n";
		expect(extractSection(md, "## Next Steps")).toBe("Content here");
	});

	it("returns empty for empty content", () => {
		expect(extractSection("", "## Anything")).toBe("");
	});
});

describe("notifyUser", () => {
	it("writes formatted message to stderr", () => {
		const writes: string[] = [];
		const orig = process.stderr.write;
		process.stderr.write = ((chunk: string | Buffer) => {
			writes.push(String(chunk));
			return true;
		}) as typeof process.stderr.write;

		try {
			notifyUser("hello %s, count: %d", "world", 42);
			expect(writes.length).toBe(1);
			expect(writes[0]).toContain("[alfred]");
			expect(writes[0]).toContain("hello world, count: 42");
		} finally {
			process.stderr.write = orig;
		}
	});

	it("handles no args", () => {
		const writes: string[] = [];
		const orig = process.stderr.write;
		process.stderr.write = ((chunk: string | Buffer) => {
			writes.push(String(chunk));
			return true;
		}) as typeof process.stderr.write;

		try {
			notifyUser("simple message");
			expect(writes[0]).toContain("simple message");
		} finally {
			process.stderr.write = orig;
		}
	});
});

describe("emitAdditionalContext", () => {
	it("writes JSON to stdout", () => {
		const writes: string[] = [];
		const orig = process.stdout.write;
		process.stdout.write = ((chunk: string | Buffer) => {
			writes.push(String(chunk));
			return true;
		}) as typeof process.stdout.write;

		try {
			emitAdditionalContext("TestEvent", "some context");
			expect(writes.length).toBe(1);
			const parsed = JSON.parse(writes[0]!);
			expect(parsed.hookSpecificOutput.hookEventName).toBe("TestEvent");
			expect(parsed.hookSpecificOutput.additionalContext).toBe("some context");
		} finally {
			process.stdout.write = orig;
		}
	});
});
