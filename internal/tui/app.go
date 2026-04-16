package tui

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/skalluru/velocix/internal/store"
)

func Run(st *store.Store, org string, logger *slog.Logger) error {
	m := NewModel(st, org)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
