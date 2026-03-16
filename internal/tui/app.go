// Package tui implements the alfred dashboard using bubbletea v2.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

const (
	tabOverview  = 0
	tabTasks     = 1
	tabKnowledge = 2
	tabActivity  = 3
	tabCount     = 4
)

var tabNames = [tabCount]string{"Overview", "Tasks", "Knowledge", "Activity"}

// specTaskGroup groups spec files by task for the Specs tab.
type specTaskGroup struct {
	Slug      string
	FileCount int
	TotalSize int64
	Files     []SpecEntry
}

// tickMsg triggers periodic data refresh.
type tickMsg time.Time

// searchResultMsg carries async semantic search results.
type searchResultMsg []KnowledgeEntry

// debounceTickMsg triggers a debounced search after typing pauses.
type debounceTickMsg struct{ seq int }

// semanticSearchResultMsg carries results from async semantic search.
type semanticSearchResultMsg struct {
	results []KnowledgeEntry
}

// dataLoadedMsg carries refreshed data from async loading.
type dataLoadedMsg struct {
	activeSlug  string
	allTasks    []TaskDetail
	specs       []SpecEntry
	knowledge   []KnowledgeEntry
	activity    []ActivityEntry
	knStats     KnowledgeStats
	epics       []EpicSummary
	decisions   []DecisionEntry
	specGroups  []specTaskGroup
	validations map[string]*spec.ValidationReport
	memHealth   MemoryHealthStats
	confStats   map[string]*spec.ConfidenceSummary
}

// Model is the root bubbletea model.
type Model struct {
	ds      DataSource
	version string
	width   int
	height  int

	// Tab state.
	activeTab int
	showHelp  bool

	// Bubbles components.
	viewport    viewport.Model
	helpModel   help.Model
	spinner     spinner.Model
	searchInput textinput.Model
	progress    progress.Model

	// Overlay (floating window) state.
	overlayActive  bool
	overlayTitle   string
	overlayVP      viewport.Model
	overlayRawMD   string   // raw markdown content for clipboard copy
	breadcrumbs    []string // navigation path shown in overlay header
	overlayCopied  bool     // flash "Copied!" message

	// Review mode state (within Specs overlay).
	reviewMode      bool              // true when reviewing a spec file
	reviewFile      string            // which file is being reviewed (e.g. "design.md")
	reviewTaskSlug  string            // which task's spec
	reviewLines     []string          // raw lines of the file
	reviewCursor    int               // current line (0-based)
	reviewComments  map[int]string    // line number → comment body (pending)
	reviewInput     textarea.Model    // multi-line comment input
	reviewInputLine int               // which line the input is for
	reviewEditing   bool              // true when typing a comment
	reviewRounds    []spec.Review     // all review rounds for the current task
	reviewRoundIdx  int               // current round index (len-1 = latest)

	// Data caches.
	activeSlug string
	allTasks   []TaskDetail
	specs      []SpecEntry
	knowledge  []KnowledgeEntry
	activity   []ActivityEntry

	// Tasks tab state.
	taskCursor int
	taskLevel  int // 0=task list, 1=spec files for selected task

	// Specs tab state (used in drill-down from Tasks).
	specGroups      []specTaskGroup
	specGroupCursor int
	specFileCursor  int
	specLevel       int // 0=groups, 1=files

	// Knowledge tab state.
	knList         list.Model
	knStats        KnowledgeStats
	promotions     []KnowledgeEntry
	decisions      []DecisionEntry
	searchBusy     bool
	searchMode     bool             // Ctrl+S semantic search active
	searchResults  []KnowledgeEntry // semantic search results

	// Activity tab state.
	activityTable  table.Model
	epics          []EpicSummary
	activityFilter string // ""=all, "spec.init", "spec.complete", "review.submit"

	// Cached data from DataSource extensions.
	validations map[string]*spec.ValidationReport
	memHealth   MemoryHealthStats
	confStats   map[string]*spec.ConfidenceSummary

	// Markdown renderer.
	mdRenderer *glamour.TermRenderer

	// Shimmer animation frame counter.
	shimmerFrame int

	// Debounce sequence for live search.
	debounceSeq int

	// Review confirmation state.
	reviewConfirmPending bool
	reviewConfirmStatus  spec.ReviewStatus
}

// New creates a new dashboard Model with version string for the title bar.
func New(ds DataSource, version string) Model {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(aqua)),
	)
	h := help.New()
	h.Styles = help.DefaultStyles(true)
	ti := textinput.New()
	ti.Placeholder = "semantic search..."
	ti.CharLimit = 200
	prog := progress.New(
		progress.WithColors(aqua, lipgloss.Color("#333")),
		progress.WithoutPercentage(),
		progress.WithWidth(20),
		progress.WithFillCharacters('#', '-'),
	)
	knl := newKnowledgeList(80, 20) // sized later in WindowSizeMsg

	// Activity timeline table — non-interactive (Blur), styled header.
	cellStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0bab0")).Padding(0, 1)
	tblStyles := table.Styles{
		Header:   lipgloss.NewStyle().Bold(true).Foreground(gold).Padding(0, 1),
		Cell:     cellStyle,
		Selected: cellStyle,
	}
	actTbl := table.New(
		table.WithColumns([]table.Column{
			{Title: "Time", Width: 6},
			{Title: "Action", Width: 12},
			{Title: "Target", Width: 24},
			{Title: "Detail", Width: 36},
		}),
		table.WithHeight(10),
		table.WithStyles(tblStyles),
	)
	actTbl.Blur()

	return Model{
		ds:            ds,
		version:       version,
		helpModel:     h,
		spinner:       sp,
		searchInput:   ti,
		progress:      prog,
		knList:        knl,
		activityTable: actTbl,
	}
}

// loadDataCmd returns a tea.Cmd that loads data asynchronously in a goroutine.
func (m *Model) loadDataCmd() tea.Cmd {
	ds := m.ds
	_ = m.searchBusy // kept for future semantic search
	return func() tea.Msg {
		done := make(chan dataLoadedMsg, 1)
		go func() {
			var msg dataLoadedMsg
			msg.activeSlug = ds.ActiveTask()
			msg.allTasks = ds.TaskDetails()

			specs := ds.Specs()
			msg.specs = specs
			groupMap := make(map[string]*specTaskGroup)
			var groupOrder []string
			for _, s := range specs {
				g, ok := groupMap[s.TaskSlug]
				if !ok {
					g = &specTaskGroup{Slug: s.TaskSlug}
					groupMap[s.TaskSlug] = g
					groupOrder = append(groupOrder, s.TaskSlug)
				}
				g.FileCount++
				g.TotalSize += s.Size
				g.Files = append(g.Files, s)
			}
			groups := make([]specTaskGroup, 0, len(groupOrder))
			for _, slug := range groupOrder {
				groups = append(groups, *groupMap[slug])
			}
			msg.specGroups = groups

			msg.activity = ds.RecentActivity(50)
			msg.knStats = ds.KnowledgeStats()
			msg.epics = ds.Epics()
			msg.decisions = ds.AllDecisions(20)
			msg.knowledge = ds.RecentKnowledge(100)

			// Validation + confidence for each active task.
			msg.validations = make(map[string]*spec.ValidationReport)
			msg.confStats = make(map[string]*spec.ConfidenceSummary)
			for _, t := range msg.allTasks {
				if r := ds.Validation(t.Slug); r != nil {
					msg.validations[t.Slug] = r
				}
				if cs := ds.ConfidenceStats(t.Slug); cs != nil {
					msg.confStats[t.Slug] = cs
				}
			}
			msg.memHealth = ds.MemoryHealth()

			done <- msg
		}()

		select {
		case result := <-done:
			return result
		case <-time.After(4 * time.Second):
			return dataLoadedMsg{}
		}
	}
}

// applyDataLoaded applies a dataLoadedMsg to the model.
func (m *Model) applyDataLoaded(msg dataLoadedMsg) {
	m.activeSlug = msg.activeSlug
	m.allTasks = msg.allTasks
	m.specs = msg.specs
	m.specGroups = msg.specGroups
	if m.taskCursor >= len(m.allTasks) {
		m.taskCursor = max(0, len(m.allTasks)-1)
	}
	if m.specGroupCursor >= len(m.specGroups) {
		m.specGroupCursor = max(0, len(m.specGroups)-1)
	}
	m.activity = msg.activity
	if len(m.activity) > 0 {
		shown := min(20, len(m.activity))
		tblW := max(60, m.width-6)
		timeW := 6
		actionW := 12
		targetW := max(16, tblW*25/100)
		detailW := max(10, tblW-timeW-actionW-targetW-8)
		rows := make([]table.Row, 0, shown)
		for i := range shown {
			a := m.activity[i]
			rows = append(rows, table.Row{
				a.Timestamp.Format("15:04"),
				formatAuditAction(a.Action),
				truncStr(a.Target, targetW),
				truncStr(a.Detail, detailW),
			})
		}
		m.activityTable.SetColumns([]table.Column{
			{Title: "Time", Width: timeW},
			{Title: "Action", Width: actionW},
			{Title: "Target", Width: targetW},
			{Title: "Detail", Width: detailW},
		})
		m.activityTable.SetWidth(tblW)
		m.activityTable.SetHeight(min(shown+1, 20))
		m.activityTable.SetRows(rows)
	}
	m.knStats = msg.knStats
	m.epics = msg.epics
	m.decisions = msg.decisions
	if msg.knowledge != nil {
		m.knowledge = msg.knowledge
		m.knList.SetItems(knowledgeEntriesToItems(m.knowledge))
	}
	m.promotions = m.promotions[:0]
	for _, k := range m.knowledge {
		if k.Source == "memory" {
			if (k.SubType == "general" && k.HitCount >= 5) || (k.SubType == "pattern" && k.HitCount >= 15) {
				m.promotions = append(m.promotions, k)
			}
		}
	}
	m.validations = msg.validations
	m.memHealth = msg.memHealth
	m.confStats = msg.confStats
	m.rebuildTasksViewport()
}

func (m *Model) switchTab(tab int) {
	m.activeTab = tab
	m.searchBusy = false
	if tab == tabTasks {
		m.taskLevel = 0
		m.rebuildTasksViewport()
	}
}

func (m Model) contentHeight() int {
	h := m.height - 5
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		shimmerCmd(),
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mdRenderer, _ = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(max(40, m.width-6)),
		)
		m.viewport = viewport.New(
			viewport.WithWidth(m.width-4),
			viewport.WithHeight(m.contentHeight()),
		)
		m.viewport.SoftWrap = true
		m.progress = progress.New(
			progress.WithColors(aqua, lipgloss.Color("#333")),
			progress.WithoutPercentage(),
			progress.WithWidth(min(20, m.width/4)),
			progress.WithFillCharacters('#', '-'),
		)
		m.knList.SetSize(m.width-4, m.contentHeight()-4)
		{
			tblW := max(60, m.width-6)
			targetW := max(16, tblW*25/100)
			detailW := max(10, tblW-6-12-targetW-8)
			m.activityTable.SetColumns([]table.Column{
				{Title: "Time", Width: 6},
				{Title: "Action", Width: 12},
				{Title: "Target", Width: targetW},
				{Title: "Detail", Width: detailW},
			})
			m.activityTable.SetWidth(tblW)
			m.activityTable.SetHeight(min(20, m.contentHeight()-8))
		}
		return m, m.loadDataCmd()

	case clipboardMsg:
		return m, nil

	case dataLoadedMsg:
		m.applyDataLoaded(msg)
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			m.loadDataCmd(),
			tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}),
		)

	case searchResultMsg:
		m.knowledge = []KnowledgeEntry(msg)
		m.searchBusy = false
		m.updateKnowledgeListItems()
		return m, nil

	case semanticSearchResultMsg:
		m.searchBusy = false
		m.searchResults = msg.results
		if len(msg.results) > 0 {
			m.knList.SetItems(knowledgeEntriesToItems(msg.results))
		} else {
			m.knList.SetItems(nil)
		}
		return m, nil

	case shimmerTickMsg:
		m.shimmerFrame++
		m.rebuildTasksViewport()
		m.rebuildTaskOverlay()
		cmds = append(cmds, shimmerCmd())
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		if m.searchBusy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyPressMsg:
		if m.overlayActive {
			return m.updateOverlay(msg)
		}
		if m.activeTab == tabKnowledge && m.knList.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.knList, cmd = m.knList.Update(msg)
			return m, cmd
		}
		if m.showHelp {
			if key.Matches(msg, keys.Help, keys.Back, keys.Quit) {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, keys.Tab):
			m.switchTab((m.activeTab + 1) % tabCount)
			return m, nil
		case key.Matches(msg, keys.BackTab):
			m.switchTab((m.activeTab - 1 + tabCount) % tabCount)
			return m, nil
		case key.Matches(msg, keys.Search):
			if m.activeTab == tabKnowledge && !m.overlayActive {
				var cmd tea.Cmd
				m.knList, cmd = m.knList.Update(msg)
				return m, cmd
			}
		case key.Matches(msg, keys.Review):
			if m.activeTab == tabTasks && m.taskLevel == 0 && !m.overlayActive {
				return m.tryDirectReview()
			}
		}

		switch m.activeTab {
		case tabOverview:
			m.viewport, _ = m.viewport.Update(msg)
		case tabTasks:
			if m.taskLevel == 0 {
				return m.updateTasksList(msg)
			}
			return m.updateSpecs(msg)
		case tabKnowledge:
			return m.updateKnowledge(msg)
		case tabActivity:
			return m.handleActivityKey(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	var content string
	if m.showHelp {
		content = "\n" + m.helpModel.FullHelpView(keys.FullHelp())
	} else {
		switch m.activeTab {
		case tabOverview:
			content = m.overviewView()
		case tabTasks:
			if m.taskLevel == 0 {
				content = m.tasksView()
			} else {
				content = m.specsView()
			}
		case tabKnowledge:
			content = m.knowledgeListView()
		case tabActivity:
			content = m.activityView()
		}
	}

	bg := lipgloss.JoinVertical(lipgloss.Left,
		m.tabBarView(),
		content,
		m.helpBar(),
	)

	var view string
	if m.overlayActive {
		view = m.renderOverlayView(bg)
	} else {
		view = bg
	}

	v := tea.NewView(stripDECRPM(view))
	v.AltScreen = true
	return v
}

func (m Model) tabBarView() string {
	title := titleBarStyle.Render("alfred dashboard")
	if m.version != "" {
		title += " " + versionStyle.Render("v"+m.version)
	}

	var tabs []string
	for i, name := range tabNames {
		label := name + m.tabBadge(i)
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	if m.searchBusy {
		tabRow += " " + m.spinner.View()
	}

	return tabBarStyle.Width(m.width).Render(
		title + "\n\n" + tabRow,
	)
}

func (m Model) tabBadge(tab int) string {
	switch tab {
	case tabTasks:
		if len(m.allTasks) == 0 {
			return ""
		}
		hasBlocker := false
		for _, t := range m.allTasks {
			if t.HasBlocker {
				hasBlocker = true
				break
			}
		}
		if hasBlocker {
			return fmt.Sprintf("(%d!)", len(m.allTasks))
		}
		return fmt.Sprintf("(%d)", len(m.allTasks))
	case tabKnowledge:
		total := m.knStats.Total
		if total == 0 {
			return ""
		}
		return fmt.Sprintf("(%d)", total)
	case tabActivity:
		if len(m.activity) == 0 {
			return ""
		}
		return fmt.Sprintf("(%d)", len(m.activity))
	default:
		return ""
	}
}

func (m Model) helpBar() string {
	return "\n" + m.helpModel.ShortHelpView(keys.ShortHelp())
}

// overviewView renders the Overview tab with project health summary.
func (m Model) overviewView() string {
	var b strings.Builder
	b.WriteString("\n")
	maxW := m.width - 6

	// Task summary with validation.
	activeCount := 0
	completedCount := 0
	for _, t := range m.allTasks {
		if t.Status == "completed" || t.Status == "done" {
			completedCount++
		} else {
			activeCount++
		}
	}
	b.WriteString("  " + sectionHeader.Render("Tasks") + "\n")
	fmt.Fprintf(&b, "  %d active, %d completed\n", activeCount, completedCount)

	// Per-task validation summary.
	if len(m.validations) > 0 {
		for _, t := range m.allTasks {
			if t.Status == "completed" || t.Status == "done" {
				continue
			}
			if r, ok := m.validations[t.Slug]; ok {
				passed := 0
				for _, c := range r.Checks {
					if c.Status == "pass" {
						passed++
					}
				}
				total := len(r.Checks)
				icon := scoreStyle.Render("ok")
				if passed < total {
					icon = blockerStyle.Render(fmt.Sprintf("%d fail", total-passed))
				}
				fmt.Fprintf(&b, "    %-20s %d/%d checks  %s\n", truncStr(t.Slug, 20), passed, total, icon)
			}
		}
	}
	b.WriteString("\n")

	// Memory health.
	b.WriteString("  " + sectionHeader.Render("Knowledge Health") + "\n")
	fmt.Fprintf(&b, "  Total: %d  |  decision: %d  pattern: %d  rule: %d  general: %d\n",
		m.knStats.Total, m.knStats.Decision, m.knStats.Pattern, m.knStats.Rule, m.knStats.General)
	if m.memHealth.Total > 0 {
		staleLabel := dimStyle.Render(fmt.Sprintf("%d", m.memHealth.StaleCount))
		if m.memHealth.StaleCount > 0 {
			staleLabel = blockerStyle.Render(fmt.Sprintf("%d stale", m.memHealth.StaleCount))
		}
		conflictLabel := dimStyle.Render("0")
		if m.memHealth.ConflictCount > 0 {
			conflictLabel = reviewCommentMarker.Render(fmt.Sprintf("%d conflicts", m.memHealth.ConflictCount))
		}
		fmt.Fprintf(&b, "  Memories: %d  |  %s  |  %s\n", m.memHealth.Total, staleLabel, conflictLabel)
	}
	b.WriteString("\n")

	// Confidence distribution across active specs.
	hasConf := false
	for _, t := range m.allTasks {
		if cs, ok := m.confStats[t.Slug]; ok && cs != nil {
			if !hasConf {
				b.WriteString("  " + sectionHeader.Render("Confidence") + "\n")
				hasConf = true
			}
			groundingStr := ""
			if len(cs.GroundingDist) > 0 {
				parts := make([]string, 0, 4)
				for _, g := range []string{"verified", "reviewed", "inferred", "speculative"} {
					if n, ok := cs.GroundingDist[g]; ok && n > 0 {
						parts = append(parts, fmt.Sprintf("%s:%d", g[:3], n))
					}
				}
				groundingStr = dimStyle.Render(strings.Join(parts, " "))
			}
			fmt.Fprintf(&b, "    %-20s avg:%.1f  low:%d/%d  %s\n",
				truncStr(t.Slug, 20), cs.Avg, cs.LowCount, cs.Total, groundingStr)
		}
	}
	if hasConf {
		b.WriteString("\n")
	}

	// Epic progress.
	if len(m.epics) > 0 {
		b.WriteString("  " + sectionHeader.Render("Epics") + "\n")
		for _, e := range m.epics {
			progBar := ""
			if e.Total > 0 {
				pct := float64(e.Completed) / float64(e.Total)
				barW := 10
				filled := int(pct * float64(barW))
				progBar = strings.Repeat("#", filled) + strings.Repeat("-", barW-filled)
				progBar += fmt.Sprintf(" %d%%", int(pct*100))
			}
			fmt.Fprintf(&b, "  %-20s %s  %s\n", truncStr(e.Name, 20), progBar, styledStatus(e.Status))
		}
		b.WriteString("\n")
	}

	// Recent decisions.
	if len(m.decisions) > 0 {
		shown := min(5, len(m.decisions))
		b.WriteString("  " + sectionHeader.Render("Recent Decisions") + "\n")
		for i := range shown {
			d := m.decisions[i]
			b.WriteString("  " + truncStr(d.Title, maxW-25) + "  " + dimStyle.Render(d.TaskSlug) + "\n")
			if d.Chosen != "" {
				b.WriteString("    " + dimStyle.Render("-> "+truncStr(d.Chosen, maxW-7)) + "\n")
			}
		}
		b.WriteString("\n")
	}

	if len(m.allTasks) == 0 && m.knStats.Total == 0 {
		b.WriteString("  " + dimStyle.Render("no tasks or knowledge yet — use dossier init to start"))
	}

	return b.String()
}

func (m Model) renderMarkdown(content string) string {
	if m.mdRenderer == nil {
		return content
	}
	rendered, err := m.mdRenderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(rendered)
}
