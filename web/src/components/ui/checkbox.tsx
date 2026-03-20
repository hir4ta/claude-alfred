import { Check } from "@animated-color-icons/lucide-react";
import { cn } from "@/lib/utils";

interface CheckboxProps {
	checked?: boolean;
	className?: string;
}

export function Checkbox({ checked, className }: CheckboxProps) {
	return (
		<div
			className={cn(
				"flex size-4 shrink-0 items-center justify-center rounded border transition-colors",
				checked
					? "border-brand-pattern bg-brand-pattern text-white"
					: "border-muted-foreground/30 bg-background",
				className,
			)}
		>
			{checked && <Check className="size-3" strokeWidth={3} />}
		</div>
	);
}
