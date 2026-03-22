import { useState } from "react";

export type ViewMode = "list" | "bookshelf";

function getStoredViewMode(tab: string, defaultMode: ViewMode): ViewMode {
	try {
		const stored = localStorage.getItem(`alfred-view-${tab}`);
		if (stored === "card") return "bookshelf"; // migrate old value
		if (stored === "list" || stored === "bookshelf") return stored;
	} catch {
		// localStorage unavailable
	}
	return defaultMode;
}

export function useViewMode(tab: string, defaultMode: ViewMode): [ViewMode, (mode: ViewMode) => void] {
	const [mode, setModeState] = useState<ViewMode>(() => getStoredViewMode(tab, defaultMode));

	const setMode = (newMode: ViewMode) => {
		setModeState(newMode);
		try {
			localStorage.setItem(`alfred-view-${tab}`, newMode);
		} catch {
			// ignore
		}
	};

	return [mode, setMode];
}
