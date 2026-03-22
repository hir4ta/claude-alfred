import { useState } from "react";

export type ViewMode = "list" | "card";

function getStoredViewMode(tab: string, defaultMode: ViewMode): ViewMode {
	try {
		const stored = localStorage.getItem(`alfred-view-${tab}`);
		if (stored === "list" || stored === "card") return stored;
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
