package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/skalluru/velocix/internal/model"
	"github.com/skalluru/velocix/internal/store"
)

type runsUpdatedMsg struct{}
type tickMsg time.Time

type filterField int

const (
	filterNone filterField = iota
	filterRepo
	filterStatus
	filterWorkflow
)

type Model struct {
	store        *store.Store
	org          string
	runs         []model.WorkflowRun
	cursor       int
	offset       int
	width        int
	height       int
	filterRepo   string
	filterStatus string
	filterWF     string
	activeFilter filterField
	repoNames    []string
	wfNames      []string
	statusOpts   []string
	repoIdx      int
	statusIdx    int
	wfIdx        int
	sub          chan struct{}
}

func NewModel(st *store.Store, org string) *Model {
	sub := st.Subscribe()
	return &Model{
		store:      st,
		org:        org,
		sub:        sub,
		statusOpts: []string{"", "success", "failure", "in_progress", "queued", "cancelled"},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.waitForUpdate(),
		m.tickCmd(),
	)
}

func (m *Model) waitForUpdate() tea.Cmd {
	return func() tea.Msg {
		<-m.sub
		return runsUpdatedMsg{}
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) refreshData() {
	m.runs = m.store.Filter(m.filterRepo, m.filterStatus, m.filterWF)
	m.repoNames = m.store.GetRepoNames()
	m.wfNames = m.store.GetWorkflowNames()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case runsUpdatedMsg:
		m.refreshData()
		return m, m.waitForUpdate()

	case tickMsg:
		m.refreshData()
		return m, m.tickCmd()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.store.Unsubscribe(m.sub)
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			if m.cursor < m.offset {
				m.offset = m.cursor
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.runs)-1 {
				m.cursor++
			}
			tableH := m.tableHeight()
			if m.cursor >= m.offset+tableH {
				m.offset = m.cursor - tableH + 1
			}

		case key.Matches(msg, keys.Open):
			if m.cursor < len(m.runs) {
				openURL(m.runs[m.cursor].HTMLURL)
			}

		case key.Matches(msg, keys.FilterRepo):
			m.repoIdx = (m.repoIdx + 1) % (len(m.repoNames) + 1)
			if m.repoIdx == 0 {
				m.filterRepo = ""
			} else {
				m.filterRepo = m.repoNames[m.repoIdx-1]
			}
			m.cursor = 0
			m.offset = 0
			m.refreshData()

		case key.Matches(msg, keys.FilterStatus):
			m.statusIdx = (m.statusIdx + 1) % len(m.statusOpts)
			m.filterStatus = m.statusOpts[m.statusIdx]
			m.cursor = 0
			m.offset = 0
			m.refreshData()

		case key.Matches(msg, keys.FilterWorkflow):
			m.wfIdx = (m.wfIdx + 1) % (len(m.wfNames) + 1)
			if m.wfIdx == 0 {
				m.filterWF = ""
			} else {
				m.filterWF = m.wfNames[m.wfIdx-1]
			}
			m.cursor = 0
			m.offset = 0
			m.refreshData()

		case key.Matches(msg, keys.ClearFilter):
			m.filterRepo = ""
			m.filterStatus = ""
			m.filterWF = ""
			m.repoIdx = 0
			m.statusIdx = 0
			m.wfIdx = 0
			m.cursor = 0
			m.offset = 0
			m.refreshData()
		}
	}
	return m, nil
}

func (m *Model) tableHeight() int {
	// header(3) + filter(1) + table header(1) + status bar(2)
	h := m.height - 7
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render(fmt.Sprintf(" ◆ Velocix  |  %s  |  %d runs", m.org, len(m.runs))))
	b.WriteString("\n")

	// Filter bar
	filters := fmt.Sprintf(" [1] Repo: %s  [2] Status: %s  [3] Workflow: %s  [c] Clear",
		filterDisplay(m.filterRepo, "All"),
		filterDisplay(m.filterStatus, "All"),
		filterDisplay(m.filterWF, "All"))
	b.WriteString(filterBarStyle.Render(filters))
	b.WriteString("\n")

	// Table header
	b.WriteString(tableHeaderStyle.Render(
		fmt.Sprintf("  %-10s %-24s %-24s %-20s %-10s %-14s %-8s %-10s",
			"STATUS", "REPO", "WORKFLOW", "BRANCH", "EVENT", "ACTOR", "RUN", "UPDATED")))
	b.WriteString("\n")

	// Table rows
	tableH := m.tableHeight()
	end := m.offset + tableH
	if end > len(m.runs) {
		end = len(m.runs)
	}

	for i := m.offset; i < end; i++ {
		r := m.runs[i]
		st := getStatus(r)
		icon := statusIcon(st)
		statusStr := fmt.Sprintf("%s %-9s", icon, st)

		branch := r.HeadBranch
		if len(branch) > 18 {
			branch = branch[:18] + ".."
		}
		wf := r.WorkflowName
		if len(wf) > 22 {
			wf = wf[:22] + ".."
		}
		repo := r.RepoName
		if len(repo) > 22 {
			repo = repo[:22] + ".."
		}
		actor := r.Actor
		if len(actor) > 12 {
			actor = actor[:12] + ".."
		}

		line := fmt.Sprintf("  %-10s %-24s %-24s %-20s %-10s %-14s #%-7d %s",
			statusStr, repo, wf, branch, r.Event, actor, r.RunNumber, timeAgo(r.UpdatedAt))

		if i == m.cursor {
			b.WriteString(selectedRowStyle.Render(line))
		} else {
			b.WriteString(rowStyle(st).Render(line))
		}
		b.WriteString("\n")
	}

	// Fill remaining space
	for i := end - m.offset; i < tableH; i++ {
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(statusBarStyle.Render(
		fmt.Sprintf(" q: quit | ↑↓: navigate | enter: open | 1/2/3: filter | c: clear filters | %d/%d",
			m.cursor+1, len(m.runs))))

	return b.String()
}

func filterDisplay(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func getStatus(r model.WorkflowRun) string {
	if r.Status == "in_progress" {
		return "in_progress"
	}
	if r.Status == "queued" {
		return "queued"
	}
	if r.Conclusion != "" {
		return r.Conclusion
	}
	return "queued"
}

func statusIcon(s string) string {
	switch s {
	case "success":
		return "✓"
	case "failure":
		return "✗"
	case "in_progress":
		return "●"
	case "queued":
		return "○"
	case "cancelled":
		return "–"
	default:
		return "?"
	}
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func openURL(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}
