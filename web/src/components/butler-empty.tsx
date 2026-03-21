import { BookOpen, CircleAlert, ListChecks, Search, Trophy } from "@animated-color-icons/lucide-react";
import { motion } from "motion/react";
import type { LucideProps } from "lucide-react";
import type { ComponentType } from "react";
import { useI18n } from "@/lib/i18n";
import type { TranslationKey } from "@/lib/i18n";
import { butlerSpring } from "@/lib/motion";

type ButlerScene = "empty-tray" | "monocle" | "bookshelf" | "concerned" | "bow";

const sceneIcons: Record<ButlerScene, { icon: ComponentType<LucideProps>; color: string }> = {
	"empty-tray": { icon: ListChecks, color: "#40513b" },
	monocle: { icon: Search, color: "#7b6b8d" },
	bookshelf: { icon: BookOpen, color: "#2d8b7a" },
	concerned: { icon: CircleAlert, color: "#e67e22" },
	bow: { icon: Trophy, color: "#628141" },
};

interface ButlerEmptyProps {
	scene: ButlerScene;
	messageKey: TranslationKey;
	className?: string;
}

export function ButlerEmpty({ scene, messageKey, className }: ButlerEmptyProps) {
	const { t } = useI18n();
	const { icon: Icon, color } = sceneIcons[scene];

	return (
		<motion.div
			className={`al-icon-wrapper flex flex-col items-center justify-center gap-3 min-h-[50vh] ${className ?? ""}`}
			initial={{ opacity: 0, y: 8 }}
			animate={{ opacity: 1, y: 0 }}
			transition={butlerSpring}
		>
			<Icon size={40} style={{ color }} />
			<p className="text-sm text-muted-foreground italic max-w-xs text-center whitespace-pre-line">
				{t(messageKey)}
			</p>
		</motion.div>
	);
}
