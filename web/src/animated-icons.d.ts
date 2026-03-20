declare module "@animated-color-icons/lucide-react" {
	import type { FC, SVGProps } from "react";

	interface AnimatedIconProps extends SVGProps<SVGSVGElement> {
		size?: number;
		color?: string;
		primaryColor?: string;
		secondaryColor?: string;
		strokeWidth?: number;
		className?: string;
		label?: string;
	}

	type AnimatedIcon = FC<AnimatedIconProps>;

	export const Activity: AnimatedIcon;
	export const Archive: AnimatedIcon;
	export const ArchiveRestore: AnimatedIcon;
	export const ArrowRight: AnimatedIcon;
	export const BookOpen: AnimatedIcon;
	export const Brain: AnimatedIcon;
	export const Calendar: AnimatedIcon;
	export const Check: AnimatedIcon;
	export const CheckCircle: AnimatedIcon;
	export const CheckCircle2: AnimatedIcon;
	export const CheckSquare: AnimatedIcon;
	export const ChevronDown: AnimatedIcon;
	export const ChevronLeft: AnimatedIcon;
	export const ChevronRight: AnimatedIcon;
	export const Circle: AnimatedIcon;
	export const CircleCheck: AnimatedIcon;
	export const CircleDashed: AnimatedIcon;
	export const CircleDot: AnimatedIcon;
	export const CirclePause: AnimatedIcon;
	export const CircleX: AnimatedIcon;
	export const Clock: AnimatedIcon;
	export const Copy: AnimatedIcon;
	export const Download: AnimatedIcon;
	export const Ellipsis: AnimatedIcon;
	export const Gavel: AnimatedIcon;
	export const Globe: AnimatedIcon;
	export const Grid3X3: AnimatedIcon;
	export const History: AnimatedIcon;
	export const LayoutDashboard: AnimatedIcon;
	export const Lightbulb: AnimatedIcon;
	export const ListChecks: AnimatedIcon;
	export const Maximize2: AnimatedIcon;
	export const MessageSquare: AnimatedIcon;
	export const MessageSquareText: AnimatedIcon;
	export const Minus: AnimatedIcon;
	export const Network: AnimatedIcon;
	export const Plus: AnimatedIcon;
	export const Search: AnimatedIcon;
	export const Shield: AnimatedIcon;
	export const Square: AnimatedIcon;
	export const X: AnimatedIcon;
	export const XCircle: AnimatedIcon;
	export const Zap: AnimatedIcon;
}
