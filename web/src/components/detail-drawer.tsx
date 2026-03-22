import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
} from "@/components/ui/sheet";

interface DetailDrawerProps {
	open: boolean;
	onClose: () => void;
	title: string;
	children: React.ReactNode;
}

export function DetailDrawer({ open, onClose, title, children }: DetailDrawerProps) {
	return (
		<Sheet open={open} onOpenChange={(isOpen) => { if (!isOpen) onClose(); }}>
			<SheetContent side="right" className="max-w-[480px] w-full overflow-y-auto">
				<SheetHeader>
					<SheetTitle className="text-base">{title}</SheetTitle>
				</SheetHeader>
				<div className="py-4 space-y-4">
					{children}
				</div>
			</SheetContent>
		</Sheet>
	);
}
