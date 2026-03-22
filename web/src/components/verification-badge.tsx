import { useI18n } from "@/lib/i18n";
import type { KnowledgeEntry } from "@/lib/types";

/**
 * Verification status badge for knowledge entries.
 * - overdue: verification_due < now (red)
 * - verified: last_verified exists AND verification_due >= now (green)
 * - pending: no last_verified or no verification_due (grey)
 */
export function VerificationBadge({ entry }: { entry: KnowledgeEntry }) {
	const { t } = useI18n();
	const due = entry.verification_due;
	const verified = entry.last_verified;

	if (!due) return null;

	const now = new Date();
	const isOverdue = new Date(due) < now;

	if (isOverdue) {
		return (
			<span className="inline-flex items-center rounded-full px-1.5 py-0.5 text-[9px] font-medium" style={{ backgroundColor: "#c0392b20", color: "#c0392b" }}>
				{t("knowledge.verification.overdue")}
			</span>
		);
	}
	if (verified) {
		return (
			<span className="inline-flex items-center rounded-full px-1.5 py-0.5 text-[9px] font-medium" style={{ backgroundColor: "#62814120", color: "#628141" }}>
				{t("knowledge.verification.verified")}
			</span>
		);
	}
	return (
		<span className="inline-flex items-center rounded-full px-1.5 py-0.5 text-[9px] font-medium text-muted-foreground bg-muted/50">
			{t("knowledge.verification.pending")}
		</span>
	);
}
