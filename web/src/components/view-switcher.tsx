// @ts-expect-error — List exported but missing from type declarations
import { List, BookOpen } from "@animated-color-icons/lucide-react";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useI18n } from "@/lib/i18n";
import type { ViewMode } from "@/lib/use-view-mode";

export function ViewSwitcher({
	current,
	onChange,
}: {
	current: ViewMode;
	onChange: (mode: ViewMode) => void;
}) {
	const { t } = useI18n();

	return (
		<ToggleGroup
			type="single"
			value={current}
			onValueChange={(v) => { if (v) onChange(v as ViewMode); }}
			size="sm"
		>
			<Tooltip>
				<TooltipTrigger asChild>
					<ToggleGroupItem value="list" aria-label={t("view.list")}>
						<List className="size-4" />
					</ToggleGroupItem>
				</TooltipTrigger>
				<TooltipContent>{t("view.list")}</TooltipContent>
			</Tooltip>
			<Tooltip>
				<TooltipTrigger asChild>
					<ToggleGroupItem value="bookshelf" aria-label={t("view.card")}>
						<BookOpen className="size-4" />
					</ToggleGroupItem>
				</TooltipTrigger>
				<TooltipContent>{t("view.card")}</TooltipContent>
			</Tooltip>
		</ToggleGroup>
	);
}
