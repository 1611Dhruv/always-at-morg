package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/always-at-morg/internal/client/connection"
)

// connectionSuccessMsg is sent when connection is established
type connectionSuccessMsg struct{}

// connectionErrorMsg is sent when connection fails
type connectionErrorMsg struct {
	err error
}

// connectionEventMsg wraps events from the connection manager
type connectionEventMsg struct {
	event connection.Event
}

// tickMsg is sent periodically for animations
type tickMsg time.Time

// connectCmd attempts to connect to the server
func connectCmd(serverURL string) tea.Cmd {
	return func() tea.Msg {
		mgr := connection.NewManager(serverURL)
		if err := mgr.Connect(); err != nil {
			return connectionErrorMsg{err: err}
		}
		return connectionSuccessMsg{}
	}
}

// tickCmd returns a command that sends tick messages for animations
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
