package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
)

// semanticSearchKey is the key binding for Ctrl+S semantic search.
var semanticSearchKey = key.NewBinding(
	key.WithKeys("ctrl+s"),
	key.WithHelp("C-s", "semantic search"),
)

func (m *Model) updateKnowledge(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Ctrl+S: toggle semantic search mode.
	if key.Matches(msg, semanticSearchKey) {
		if m.searchMode {
			// Exit search mode, restore normal list.
			m.searchMode = false
			m.searchResults = nil
			m.searchInput.Blur()
			m.updateKnowledgeListItems()
		} else {
			// Enter search mode.
			m.searchMode = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
		}
		return m, nil
	}

	// In search mode: handle input.
	if m.searchMode {
		if key.Matches(msg, keys.Back) {
			// Esc: exit search mode.
			m.searchMode = false
			m.searchResults = nil
			m.searchInput.Blur()
			m.updateKnowledgeListItems()
			return m, nil
		}
		if key.Matches(msg, keys.Enter) {
			// Execute semantic search.
			query := m.searchInput.Value()
			if query != "" {
				m.searchBusy = true
				return m, m.executeSemanticSearch(query)
			}
			return m, nil
		}
		// Forward to text input.
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	// Space: toggle enabled/disabled.
	if key.Matches(msg, knowledgeToggleKey) {
		idx := m.knList.Index()
		if idx < len(m.knowledge) {
			k := &m.knowledge[idx]
			if k.ID > 0 {
				newState := !k.Enabled
				if err := m.ds.ToggleEnabled(k.ID, newState); err == nil {
					m.syncKnowledgeItemEnabled(idx, newState)
				}
			}
		}
		return m, nil
	}

	// Enter: open detail overlay.
	if key.Matches(msg, keys.Enter) {
		if item := m.knList.SelectedItem(); item != nil {
			ki := item.(knowledgeItem)
			title := extractKnowledgeTitle(ki.entry)
			// Strip the leading "# title" from body to avoid triple repetition
			// (breadcrumb + overlay title + body heading).
			rawMD := renderKnowledgeDetailNoTitle(ki.entry)
			m.openOverlay(
				title,
				m.renderMarkdown(rawMD),
				"Knowledge", ki.entry.Source,
			)
			// Store raw markdown with title for clipboard copy.
			m.overlayRawMD = "# " + title + "\n\n" + rawMD
		}
		return m, nil
	}

	// Delegate all other keys (up/down/filter/pgup/pgdn) to the list.
	var cmd tea.Cmd
	m.knList, cmd = m.knList.Update(msg)
	return m, cmd
}

// executeSemanticSearch runs SemanticSearch asynchronously.
func (m *Model) executeSemanticSearch(query string) tea.Cmd {
	ds := m.ds
	return func() tea.Msg {
		results := ds.SemanticSearch(query, 20)
		return semanticSearchResultMsg{results: results}
	}
}

func extractKnowledgeTitle(k KnowledgeEntry) string {
	if k.Structured != "" {
		var raw map[string]any
		if json.Unmarshal([]byte(k.Structured), &raw) == nil {
			if t, ok := raw["title"].(string); ok && t != "" {
				return t
			}
			if t, ok := raw["text"].(string); ok && t != "" {
				return t
			}
		}
	}
	title, _ := simplifyKnowledgeLabel(k.Label)
	if len(title) < 5 {
		title = knowledgeTitle(k.Content)
	}
	return title
}

// renderKnowledgeDetailNoTitle renders a knowledge entry without the leading
// heading, since the overlay already shows the title in the header bar.
func renderKnowledgeDetailNoTitle(k KnowledgeEntry) string {
	full := renderKnowledgeDetail(k)
	// Strip leading heading line (any level: #, ##, ###) if present.
	if strings.HasPrefix(full, "#") {
		if _, after, found := strings.Cut(full, "\n"); found {
			return strings.TrimLeft(after, "\n")
		}
	}
	return full
}

// renderKnowledgeDetail renders a knowledge entry for the overlay,
// using structured data fields when available.
func renderKnowledgeDetail(k KnowledgeEntry) string {
	if k.Structured == "" {
		return formatKnowledgeContent(k.Content)
	}

	var raw map[string]any
	if json.Unmarshal([]byte(k.Structured), &raw) != nil {
		return k.Content
	}

	var b strings.Builder

	// writeSection writes a markdown section with proper spacing.
	// Ensures a blank line between heading and content, and between
	// prose text and bullet lists within the content.
	writeSection := func(heading, body string) {
		b.WriteString("## " + heading + "\n\n")
		b.WriteString(ensureListSpacing(body))
		b.WriteString("\n\n")
	}

	switch k.SubType {
	case "decision":
		if v, _ := raw["title"].(string); v != "" {
			b.WriteString("# " + v + "\n\n")
		}
		if v, _ := raw["context"].(string); v != "" {
			writeSection("Context", v)
		}
		if v, _ := raw["decision"].(string); v != "" {
			writeSection("Decision", v)
		}
		if v, _ := raw["reasoning"].(string); v != "" {
			writeSection("Reasoning", v)
		}
		if alts, ok := raw["alternatives"].([]any); ok && len(alts) > 0 {
			b.WriteString("## Alternatives\n\n")
			for _, a := range alts {
				b.WriteString("- " + fmt.Sprint(a) + "\n")
			}
			b.WriteString("\n")
		}
		if v, _ := raw["status"].(string); v != "" {
			b.WriteString("---\n\n**Status:** " + v + "\n")
		}

	case "pattern":
		if v, _ := raw["title"].(string); v != "" {
			b.WriteString("# " + v + "\n\n")
		}
		if v, _ := raw["context"].(string); v != "" {
			writeSection("Context", v)
		}
		if v, _ := raw["pattern"].(string); v != "" {
			writeSection("Pattern", v)
		}
		if v, _ := raw["applicationConditions"].(string); v != "" {
			writeSection("When to Apply", v)
		}
		if v, _ := raw["expectedOutcomes"].(string); v != "" {
			writeSection("Expected Outcomes", v)
		}
		if v, _ := raw["status"].(string); v != "" {
			b.WriteString("---\n\n**Status:** " + v + "\n")
		}

	case "rule":
		if v, _ := raw["text"].(string); v != "" {
			b.WriteString("# " + v + "\n\n")
		}
		if v, _ := raw["category"].(string); v != "" {
			b.WriteString("**Category:** " + v + "\n\n")
		}
		if v, _ := raw["priority"].(string); v != "" {
			b.WriteString("**Priority:** " + v + "\n\n")
		}
		if v, _ := raw["rationale"].(string); v != "" {
			writeSection("Rationale", v)
		}
		if v, _ := raw["status"].(string); v != "" {
			b.WriteString("---\n\n**Status:** " + v + "\n")
		}

	default:
		return formatKnowledgeContent(k.Content)
	}

	if b.Len() == 0 {
		return k.Content
	}
	return b.String()
}

// ensureListSpacing inserts a blank line before a bullet list
// that immediately follows a non-blank, non-bullet line.
func ensureListSpacing(s string) string {
	lines := strings.Split(s, "\n")
	var out strings.Builder
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(line, "- ") {
			prev := lines[i-1]
			if prev != "" && !strings.HasPrefix(prev, "- ") {
				out.WriteByte('\n')
			}
		}
		out.WriteString(line)
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// knowledgeTitle extracts a human-readable title from knowledge content.
func knowledgeTitle(content string) string {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var ch struct {
			Goal string `json:"goal"`
		}
		if json.Unmarshal([]byte(trimmed), &ch) == nil && ch.Goal != "" {
			return ch.Goal
		}
	}
	return firstContentLine(content)
}

// formatKnowledgeContent renders knowledge content for the overlay.
func formatKnowledgeContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return content
	}
	var ch map[string]any
	if json.Unmarshal([]byte(trimmed), &ch) != nil {
		return content
	}
	var sb strings.Builder
	if v, ok := ch["goal"].(string); ok && v != "" {
		sb.WriteString("## Goal\n" + v + "\n\n")
	}
	if v, ok := ch["status"].(string); ok {
		sb.WriteString("**Status:** " + v + "\n\n")
	}
	if v, ok := ch["decisions"].([]any); ok && len(v) > 0 {
		sb.WriteString("## Decisions\n")
		for _, d := range v {
			sb.WriteString("- " + fmt.Sprint(d) + "\n")
		}
		sb.WriteString("\n")
	}
	if v, ok := ch["modified_files"].([]any); ok && len(v) > 0 {
		sb.WriteString("## Modified Files\n")
		for _, f := range v {
			sb.WriteString("- " + fmt.Sprint(f) + "\n")
		}
		sb.WriteString("\n")
	}
	if sb.Len() == 0 {
		return content
	}
	return sb.String()
}
