import { TooltipProvider } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { QueryClient } from "@tanstack/react-query";
import { Link, Outlet, createRootRouteWithContext } from "@tanstack/react-router";
import { Activity, BookOpen, LayoutDashboard, ListChecks } from "lucide-react";

interface RouterContext {
	queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<RouterContext>()({
	component: RootLayout,
});

const tabs = [
	{ to: "/", label: "Overview", icon: LayoutDashboard },
	{ to: "/tasks", label: "Tasks", icon: ListChecks },
	{ to: "/knowledge", label: "Knowledge", icon: BookOpen },
	{ to: "/activity", label: "Activity", icon: Activity },
] as const;

function RootLayout() {
	return (
		<TooltipProvider>
			<div className="flex min-h-screen flex-col bg-background">
				<header className="sticky top-0 z-50 border-b border-border/60 bg-card/90 backdrop-blur-md">
					<div className="mx-auto flex h-14 max-w-7xl items-center gap-8 px-6">
						<span
							className="text-base font-semibold tracking-tight"
							style={{ fontFamily: "var(--font-display)", color: "#40513b" }}
						>
							alfred
						</span>
						<nav className="flex items-center gap-1">
							{tabs.map((tab) => {
								const Icon = tab.icon;
								return (
									<Link
										key={tab.to}
										to={tab.to}
										activeOptions={{ exact: tab.to === "/" }}
										className={cn(
											"flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all",
											"text-muted-foreground hover:text-foreground hover:bg-accent/60",
										)}
										activeProps={{
											className: "bg-accent text-foreground shadow-sm",
										}}
									>
										<Icon className="size-4" />
										{tab.label}
									</Link>
								);
							})}
						</nav>
					</div>
				</header>
				<main className="mx-auto w-full max-w-7xl flex-1 px-6 py-6">
					<Outlet />
				</main>
			</div>
		</TooltipProvider>
	);
}
