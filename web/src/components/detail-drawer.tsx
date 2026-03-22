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
	/** Use wider "open book" layout */
	bookLayout?: boolean;
}

export function DetailDrawer({ open, onClose, title, children, bookLayout }: DetailDrawerProps) {
	return (
		<Sheet open={open} onOpenChange={(isOpen) => { if (!isOpen) onClose(); }}>
			<SheetContent
				side="right"
				className={`${bookLayout ? "max-w-[720px]" : "max-w-[480px]"} w-full overflow-y-auto`}
			>
				{/* Bookmark ornament */}
				{bookLayout && (
					<div
						className="absolute top-0 right-16 w-6 h-16 rounded-b-sm z-10 flex items-end justify-center pb-1.5"
						style={{ backgroundColor: "#c0392b", boxShadow: "0 4px 8px rgba(0,0,0,0.15)" }}
					>
						<span className="text-white text-[10px]">&#9733;</span>
					</div>
				)}
				<SheetHeader>
					<SheetTitle className="text-base" style={bookLayout ? { fontFamily: "var(--font-display)" } : undefined}>
						{title}
					</SheetTitle>
				</SheetHeader>
				{bookLayout ? (
					<div className="py-4 space-y-4 relative">
						{/* Paper texture gradient */}
						<div className="absolute top-0 right-0 w-12 h-full bg-gradient-to-l from-black/[0.02] to-transparent pointer-events-none" />
						{children}
						{/* Page number decoration */}
						<div className="text-[10px] text-muted-foreground/30 text-right pr-2 pt-4 font-mono">
							FOLIO {Math.floor(Math.random() * 100) + 1}
						</div>
					</div>
				) : (
					<div className="py-4 space-y-4">
						{children}
					</div>
				)}
			</SheetContent>
		</Sheet>
	);
}
