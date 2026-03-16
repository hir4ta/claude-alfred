import type { QueryClient } from "@tanstack/react-query";

function safeParse<T>(data: string): T | null {
	try {
		return JSON.parse(data) as T;
	} catch {
		return null;
	}
}

export function setupSSE(queryClient: QueryClient): () => void {
	const es = new EventSource("/api/events");

	// Filesystem change detected — refresh everything.
	es.addEventListener("refresh", () => {
		queryClient.invalidateQueries();
	});

	// Targeted events from API mutation handlers.
	es.addEventListener("review_submitted", (e) => {
		const data = safeParse<{ slug: string }>(e.data);
		if (!data) return;
		queryClient.invalidateQueries({ queryKey: ["review", data.slug] });
		queryClient.invalidateQueries({ queryKey: ["tasks"] });
	});

	es.addEventListener("memory_changed", () => {
		queryClient.invalidateQueries({ queryKey: ["knowledge"] });
		queryClient.invalidateQueries({ queryKey: ["health"] });
	});

	es.onerror = () => {
		if (es.readyState === EventSource.CLOSED) {
			console.warn("[sse] connection closed, will not auto-reconnect");
		}
	};

	return () => es.close();
}
