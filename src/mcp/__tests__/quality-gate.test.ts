import { describe, expect, it } from "vitest";
import { checkActionability, qualityGate } from "../quality-gate.js";

describe("checkActionability", () => {
	describe("decision", () => {
		it("returns null when reasoning contains action words", () => {
			const result = checkActionability(
				{ title: "認証にセッションCookieを採用", reasoning: "XSSリスク軽減のため使用する" },
				"decision",
			);
			expect(result).toBeNull();
		});

		it("returns warning when no action words", () => {
			const result = checkActionability(
				{ title: "TypeScriptはJSのスーパーセット", reasoning: "型システムの特徴" },
				"decision",
			);
			expect(result).not.toBeNull();
			expect(result!.type).toBe("low_actionability");
		});
	});

	describe("pattern", () => {
		it("returns null when pattern contains solution words", () => {
			const result = checkActionability(
				{ title: "大規模リファクタ前にgrep", pattern: "問題の解決: grep -r で全参照を列挙してから避ける" },
				"pattern",
			);
			expect(result).toBeNull();
		});

		it("returns warning for description-only pattern", () => {
			const result = checkActionability(
				{ title: "TypeScriptの特徴", pattern: "型システムの概要説明" },
				"pattern",
			);
			expect(result).not.toBeNull();
			expect(result!.type).toBe("low_actionability");
		});
	});

	describe("rule", () => {
		it("returns null when text contains imperative", () => {
			const result = checkActionability(
				{ title: "テストではモックDBではなく実DBを使用する", text: "テストでは常に実DBを使用すること" },
				"rule",
			);
			expect(result).toBeNull();
		});

		it("returns null for English action words", () => {
			const result = checkActionability(
				{ title: "Use real DB in tests", text: "You must always use a real database" },
				"rule",
			);
			expect(result).toBeNull();
		});

		it("returns null for negative action words", () => {
			const result = checkActionability(
				{ title: "shadow禁止", text: "shadow-sm, shadow-md を避ける" },
				"rule",
			);
			expect(result).toBeNull();
		});

		it("returns null for conditional words", () => {
			const result = checkActionability(
				{ title: "hook timeout", text: "Voyage APIがタイムアウトした場合はスキップ" },
				"rule",
			);
			expect(result).toBeNull();
		});

		it("returns warning for vague rule", () => {
			const result = checkActionability(
				{ title: "コード品質", text: "品質が重要である" },
				"rule",
			);
			expect(result).not.toBeNull();
		});
	});

	describe("edge cases", () => {
		it("returns null for unknown sub_type", () => {
			const result = checkActionability({ title: "test" }, "snapshot");
			expect(result).toBeNull();
		});

		it("handles empty fields", () => {
			const result = checkActionability({}, "decision");
			expect(result).not.toBeNull();
			expect(result!.type).toBe("low_actionability");
		});

		it("detects Japanese positive patterns", () => {
			for (const word of ["使う", "採用", "推奨", "必須", "統一", "設定", "移行"]) {
				const result = checkActionability({ title: `テスト${word}`, text: "" }, "rule");
				expect(result).toBeNull();
			}
		});

		it("detects English conditional patterns", () => {
			const result = checkActionability(
				{ title: "timeout handling", text: "when the API fails, retry" },
				"rule",
			);
			expect(result).toBeNull();
		});
	});
});

describe("qualityGate", () => {
	it("returns only actionability warning when emb is null (NFR-2)", async () => {
		const result = await qualityGate(
			null as any, // store not needed when emb is null
			null, // no embedder
			"test text",
			'{"title":"test"}',
			"decision",
			{ title: "概要説明", reasoning: "理由の記述" },
		);
		// No duplicate/contradiction checks without embedder
		expect(result.embedding).toBeNull();
		expect(result.similarExisting).toEqual([]);
		// Actionability check still runs — "概要説明" + "理由の記述" has no action words
		expect(result.warnings.some((w) => w.type === "low_actionability")).toBe(true);
	});

	it("returns no warnings when emb is null and actionable (NFR-2)", async () => {
		const result = await qualityGate(
			null as any,
			null,
			"test text",
			'{"title":"test"}',
			"rule",
			{ title: "テストでは実DBを使用する", text: "常に実DBを使用すること" },
		);
		expect(result.warnings).toEqual([]);
		expect(result.embedding).toBeNull();
	});

	it("handles embedder timeout gracefully (NFR-1)", async () => {
		const slowEmbedder = {
			model: "test-model",
			embedForStorage: () => new Promise<number[]>((resolve) => setTimeout(() => resolve([1, 2, 3]), 5000)),
		};
		const result = await qualityGate(
			null as any, // store won't be reached due to timeout
			slowEmbedder as any,
			"test",
			'{"title":"test"}',
			"rule",
			{ title: "テスト設定", text: "設定すること" },
		);
		// Should timeout and fall back — no embedding, no duplicate warnings
		expect(result.embedding).toBeNull();
		// Only actionability (which passes here)
		expect(result.warnings.filter((w) => w.type === "near_duplicate")).toEqual([]);
	}, 6000);
});
