package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
	"github.com/tessro/riff/internal/spotify/player"
	"github.com/tessro/riff/internal/tui/components"
	"github.com/tessro/riff/internal/tui/styles"
)

// Panel represents which panel is focused
type Panel int

const (
	PanelNowPlaying Panel = iota
	PanelQueue
	PanelDevices
	PanelHistory
)

// App holds the TUI application state
type App struct {
	spotifyClient *client.Client
	player        *player.Player
	refreshRate   time.Duration
}

// NewApp creates a new TUI application
func NewApp(clientID string, refreshRate time.Duration) (*App, error) {
	storage, err := auth.NewTokenStorage("")
	if err != nil {
		return nil, err
	}

	spotifyClient := client.New(clientID, storage)
	if err := spotifyClient.LoadToken(); err != nil {
		return nil, err
	}

	return &App{
		spotifyClient: spotifyClient,
		player:        player.New(spotifyClient),
		refreshRate:   refreshRate,
	}, nil
}

// Model is the main TUI model
type Model struct {
	app          *App
	width        int
	height       int
	focusedPanel Panel

	// State
	state    *core.PlaybackState
	queue    *core.Queue
	devices  []core.Device
	history  []components.HistoryEntry

	// Components
	nowPlaying *components.NowPlaying
	queueView  *components.Queue
	devicesView *components.Devices
	historyView *components.History

	// Overlays
	showHelp   bool
	showSearch bool
	searchInput string
	searchResults []components.SearchResult

	// Error handling
	lastError error

	// Quit flag
	quitting bool
}

// NewModel creates a new TUI model
func NewModel(app *App) Model {
	return Model{
		app:          app,
		focusedPanel: PanelNowPlaying,
		nowPlaying:   components.NewNowPlaying(),
		queueView:    components.NewQueue(),
		devicesView:  components.NewDevices(),
		historyView:  components.NewHistory(),
		history:      make([]components.HistoryEntry, 0),
	}
}

// Messages
type tickMsg time.Time
type stateMsg *core.PlaybackState
type queueMsg *core.Queue
type devicesMsg []core.Device
type errMsg error

// Commands
func (m Model) tick() tea.Cmd {
	return tea.Tick(m.app.refreshRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) fetchState() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		state, err := m.app.player.GetState(ctx)
		if err != nil {
			return errMsg(err)
		}
		return stateMsg(state)
	}
}

func (m Model) fetchQueue() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		queue, err := m.app.player.GetQueue(ctx)
		if err != nil {
			return errMsg(err)
		}
		return queueMsg(queue)
	}
}

func (m Model) fetchDevices() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		devices, err := m.app.player.GetDevices(ctx)
		if err != nil {
			return errMsg(err)
		}
		return devicesMsg(devices)
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.tick(),
		m.fetchState(),
		m.fetchQueue(),
		m.fetchDevices(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.tick(), m.fetchState())

	case stateMsg:
		m.lastError = nil
		oldTrack := ""
		if m.state != nil && m.state.Track != nil {
			oldTrack = m.state.Track.URI
		}
		m.state = msg

		// Track history on track change
		if m.state != nil && m.state.Track != nil && m.state.Track.URI != oldTrack {
			m.addToHistory(m.state.Track)
		}
		return m, nil

	case queueMsg:
		m.lastError = nil
		m.queue = msg
		return m, nil

	case devicesMsg:
		m.lastError = nil
		m.devices = msg
		return m, nil

	case errMsg:
		m.lastError = msg
		return m, nil

	case refreshAfterActionMsg:
		return m, tea.Batch(m.fetchState(), m.fetchQueue())
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (always work)
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "/":
		m.showSearch = true
		m.searchInput = ""
		return m, nil

	case "esc":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.showSearch {
			m.showSearch = false
			return m, nil
		}

	case "tab":
		m.focusedPanel = (m.focusedPanel + 1) % 4
		return m, nil

	case "shift+tab":
		m.focusedPanel = (m.focusedPanel + 3) % 4
		return m, nil
	}

	// Overlays capture all other keys
	if m.showHelp || m.showSearch {
		return m, nil
	}

	// Playback controls
	switch msg.String() {
	case " ":
		return m, m.togglePlayPause()
	case "n":
		return m, m.nextTrack()
	case "p":
		return m, m.prevTrack()
	case "+", "=":
		return m, m.volumeUp()
	case "-":
		return m, m.volumeDown()
	case "r":
		return m, tea.Batch(m.fetchState(), m.fetchQueue(), m.fetchDevices())
	}

	// Panel-specific keys
	switch m.focusedPanel {
	case PanelQueue:
		switch msg.String() {
		case "j", "down":
			m.queueView.ScrollDown()
		case "k", "up":
			m.queueView.ScrollUp()
		case "enter":
			// Could implement play from queue position
		}
	case PanelDevices:
		switch msg.String() {
		case "j", "down":
			m.devicesView.SelectNext()
		case "k", "up":
			m.devicesView.SelectPrev()
		case "enter":
			return m, m.transferToDevice()
		}
	}

	return m, nil
}

func (m Model) togglePlayPause() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if m.state != nil && m.state.IsPlaying {
			m.app.player.Pause(ctx)
		} else {
			m.app.player.Play(ctx)
		}
		return nil
	}
}

type refreshAfterActionMsg struct{}

func (m Model) nextTrack() tea.Cmd {
	return func() tea.Msg {
		m.app.player.Next(context.Background())
		// Small delay to let Spotify update state
		time.Sleep(200 * time.Millisecond)
		return refreshAfterActionMsg{}
	}
}

func (m Model) prevTrack() tea.Cmd {
	return func() tea.Msg {
		m.app.player.Prev(context.Background())
		// Small delay to let Spotify update state
		time.Sleep(200 * time.Millisecond)
		return refreshAfterActionMsg{}
	}
}

func (m Model) volumeUp() tea.Cmd {
	return func() tea.Msg {
		if m.state != nil {
			newVol := m.state.Volume + 5
			if newVol > 100 {
				newVol = 100
			}
			m.app.player.Volume(context.Background(), newVol)
		}
		return nil
	}
}

func (m Model) volumeDown() tea.Cmd {
	return func() tea.Msg {
		if m.state != nil {
			newVol := m.state.Volume - 5
			if newVol < 0 {
				newVol = 0
			}
			m.app.player.Volume(context.Background(), newVol)
		}
		return nil
	}
}

func (m Model) transferToDevice() tea.Cmd {
	return func() tea.Msg {
		selected := m.devicesView.Selected()
		if selected >= 0 && selected < len(m.devices) {
			device := m.devices[selected]
			m.app.player.TransferPlayback(context.Background(), device.ID, true)
		}
		return nil
	}
}

func (m *Model) addToHistory(track *core.Track) {
	entry := components.HistoryEntry{
		Track:    track,
		PlayedAt: time.Now(),
	}

	// Add to front, keep max 50 entries
	m.history = append([]components.HistoryEntry{entry}, m.history...)
	if len(m.history) > 50 {
		m.history = m.history[:50]
	}
}

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Show overlays if active
	if m.showHelp {
		return m.renderHelp()
	}

	if m.showSearch {
		return m.renderSearch()
	}

	// Main layout: two columns
	// Left: Now Playing (top), Queue (bottom)
	// Right: Devices (top), History (bottom)

	leftWidth := m.width * 60 / 100
	rightWidth := m.width - leftWidth - 2
	topHeight := m.height * 40 / 100
	bottomHeight := m.height - topHeight - 2

	// Render panels
	nowPlaying := m.nowPlaying.Render(m.state, leftWidth-2, topHeight-2, m.focusedPanel == PanelNowPlaying)
	queueView := m.queueView.Render(m.queue, leftWidth-2, bottomHeight-2, m.focusedPanel == PanelQueue)
	devicesView := m.devicesView.Render(m.devices, rightWidth-2, topHeight-2, m.focusedPanel == PanelDevices)
	historyView := m.historyView.Render(m.history, rightWidth-2, bottomHeight-2, m.focusedPanel == PanelHistory)

	// Compose layout
	leftCol := lipgloss.JoinVertical(lipgloss.Left, nowPlaying, queueView)
	rightCol := lipgloss.JoinVertical(lipgloss.Left, devicesView, historyView)

	main := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	// Status bar
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

func (m Model) renderStatusBar() string {
	status := styles.Dim.Render("q:quit  ?:help  /:search  space:play/pause  n:next  p:prev  +/-:volume  tab:switch panel")

	if m.lastError != nil {
		status = styles.Paused.Render("Error: " + m.lastError.Error())
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Render(status)
}

func (m Model) renderHelp() string {
	help := `
  Riff TUI - Keyboard Shortcuts
  ═══════════════════════════════

  Global
  ──────
  q, Ctrl+C    Quit
  ?            Toggle help
  /            Search
  Tab          Next panel
  Shift+Tab    Previous panel
  r            Refresh

  Playback
  ────────
  Space        Play/Pause
  n            Next track
  p            Previous track
  +/=          Volume up
  -            Volume down

  Queue Panel
  ───────────
  j/↓          Scroll down
  k/↑          Scroll up
  Enter        Play selected

  Devices Panel
  ─────────────
  j/↓          Select next
  k/↑          Select previous
  Enter        Transfer playback

  Press ? or Esc to close
`

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(styles.BorderStyle.Render(help))
}

func (m Model) renderSearch() string {
	search := lipgloss.NewStyle().
		Width(60).
		Padding(1, 2).
		Render("Search: " + m.searchInput + "█\n\nPress Esc to cancel")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(styles.FocusedBorder.Render(search))
}

// Run starts the TUI application
func Run(clientID string, refreshRate time.Duration) error {
	app, err := NewApp(clientID, refreshRate)
	if err != nil {
		return err
	}

	model := NewModel(app)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err = p.Run()
	return err
}
