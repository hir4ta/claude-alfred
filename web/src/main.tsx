import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { I18nProvider } from "@/lib/i18n";
import { setupSSE } from "@/lib/sse";
import { routeTree } from "./routeTree.gen";
import "@/styles/globals.css";

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			retry: 1,
			refetchOnWindowFocus: true,
		},
	},
});

const router = createRouter({
	routeTree,
	context: { queryClient },
	defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
	interface Register {
		router: typeof router;
	}
}

// SSE → query invalidation
const cleanupSSE = setupSSE(queryClient);
window.addEventListener("beforeunload", cleanupSSE);

createRoot(document.getElementById("root")!).render(
	<StrictMode>
		<I18nProvider>
			<QueryClientProvider client={queryClient}>
				<RouterProvider router={router} />
			</QueryClientProvider>
		</I18nProvider>
	</StrictMode>,
);
