package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/textinput"
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

// SearchType represents the type of search to perform
type SearchType int

const (
	SearchAll SearchType = iota
	SearchTracks
	SearchAlbums
	SearchArtists
	SearchPlaylists
)

// searchResult represents a search result item
type searchResult struct {
	URI       string
	ArtistURI string // For starting radio (artist context)
	Title     string
	Subtitle  string
	Type      SearchType
}

const searchDebounce = 300 * time.Millisecond

// App holds the TUI application state
type App struct {
	spotifyClient *client.Client
	player        *player.Player
	refreshRate   time.Duration
	defaultDevice string // Device name from config
}

// NewApp creates a new TUI application
func NewApp(clientID string, refreshRate time.Duration, defaultDevice string) (*App, error) {
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
		defaultDevice: defaultDevice,
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
	showHelp bool

	// Search state
	showSearch    bool
	searchInput   textinput.Model
	searchResults []searchResult
	searchCursor  int
	searchType    SearchType
	searching     bool
	lastQuery     string
	searchErr     error

	// Error handling
	lastError   error
	errorExpiry time.Time // When to clear the error

	// Quit flag
	quitting bool
}

// NewModel creates a new TUI model
func NewModel(app *App) Model {
	ti := textinput.New()
	ti.Placeholder = "Search tracks, albums, artists, playlists..."
	ti.CharLimit = 100
	ti.Width = 50

	return Model{
		app:          app,
		focusedPanel: PanelNowPlaying,
		nowPlaying:   components.NewNowPlaying(),
		queueView:    components.NewQueue(),
		devicesView:  components.NewDevices(),
		historyView:  components.NewHistory(),
		history:      make([]components.HistoryEntry, 0),
		searchInput:  ti,
	}
}

// Messages
type tickMsg time.Time
type stateMsg *core.PlaybackState
type queueMsg *core.Queue
type devicesMsg []core.Device
type historyMsg []core.HistoryEntry
type errMsg error
type defaultDeviceSetMsg string // Device name that was set as default

// Search messages
type searchDebounceMsg struct{ query string }
type searchResultsMsg struct {
	results []searchResult
	err     error
}

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

func (m Model) fetchHistory() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		history, err := m.app.player.GetRecentlyPlayed(ctx, 20)
		if err != nil {
			return errMsg(err)
		}
		return historyMsg(history)
	}
}

func (m Model) doSearch(query string) tea.Cmd {
	searchType := m.searchType
	return func() tea.Msg {
		if query == "" {
			return searchResultsMsg{results: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Determine which types to search based on searchType
		var types []client.SearchType
		switch searchType {
		case SearchTracks:
			types = []client.SearchType{client.SearchTypeTrack}
		case SearchAlbums:
			types = []client.SearchType{client.SearchTypeAlbum}
		case SearchArtists:
			types = []client.SearchType{client.SearchTypeArtist}
		case SearchPlaylists:
			types = []client.SearchType{client.SearchTypePlaylist}
		default:
			types = []client.SearchType{
				client.SearchTypeTrack,
				client.SearchTypeAlbum,
				client.SearchTypeArtist,
				client.SearchTypePlaylist,
			}
		}

		resp, err := m.app.spotifyClient.Search(ctx, client.SearchOptions{
			Query: query,
			Types: types,
			Limit: 10,
		})
		if err != nil {
			return searchResultsMsg{err: err}
		}

		// Convert to searchResult slice
		var results []searchResult

		if resp.Tracks != nil {
			for _, t := range resp.Tracks.Items {
				artists := make([]string, len(t.Artists))
				for i, a := range t.Artists {
					artists[i] = a.Name
				}
				artistURI := ""
				if len(t.Artists) > 0 {
					artistURI = t.Artists[0].URI
				}
				results = append(results, searchResult{
					URI:       t.URI,
					ArtistURI: artistURI,
					Title:     t.Name,
					Subtitle:  strings.Join(artists, ", "),
					Type:      SearchTracks,
				})
			}
		}
		if resp.Albums != nil {
			for _, a := range resp.Albums.Items {
				artists := make([]string, len(a.Artists))
				for i, art := range a.Artists {
					artists[i] = art.Name
				}
				artistURI := ""
				if len(a.Artists) > 0 {
					artistURI = a.Artists[0].URI
				}
				results = append(results, searchResult{
					URI:       a.URI,
					ArtistURI: artistURI,
					Title:     a.Name,
					Subtitle:  strings.Join(artists, ", ") + " (Album)",
					Type:      SearchAlbums,
				})
			}
		}
		if resp.Artists != nil {
			for _, a := range resp.Artists.Items {
				results = append(results, searchResult{
					URI:       a.URI,
					ArtistURI: a.URI, // Artist's own URI for radio
					Title:     a.Name,
					Subtitle:  "(Artist)",
					Type:      SearchArtists,
				})
			}
		}
		if resp.Playlists != nil {
			for _, p := range resp.Playlists.Items {
				results = append(results, searchResult{
					URI:      p.URI,
					Title:    p.Name,
					Subtitle: "by " + p.Owner.DisplayName + " (Playlist)",
					Type:     SearchPlaylists,
				})
			}
		}

		return searchResultsMsg{results: results}
	}
}

func (m Model) playSearchResult(result searchResult) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		switch result.Type {
		case SearchTracks:
			_ = m.app.player.PlayURI(ctx, result.URI)
		case SearchAlbums, SearchArtists, SearchPlaylists:
			_ = m.app.player.PlayContext(ctx, result.URI, 0)
		}
		time.Sleep(200 * time.Millisecond)
		return refreshAfterActionMsg{}
	}
}

func (m Model) queueSearchResult(result searchResult) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_ = m.app.player.AddToQueue(ctx, result.URI)
		time.Sleep(200 * time.Millisecond)
		return refreshAfterActionMsg{}
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.tick(),
		m.fetchState(),
		m.fetchQueue(),
		m.fetchDevices(),
		m.fetchHistory(),
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
		if time.Now().After(m.errorExpiry) {
			m.lastError = nil
		}
		oldTrack := ""
		if m.state != nil && m.state.Track != nil {
			oldTrack = m.state.Track.URI
		}
		m.state = msg

		// On track change, update history and refresh queue
		newTrack := ""
		if m.state != nil && m.state.Track != nil {
			newTrack = m.state.Track.URI
		}
		if newTrack != oldTrack {
			if m.state != nil && m.state.Track != nil {
				m.addToHistory(m.state.Track)
			}
			return m, m.fetchQueue()
		}
		return m, nil

	case queueMsg:
		if time.Now().After(m.errorExpiry) {
			m.lastError = nil
		}
		m.queue = msg
		return m, nil

	case devicesMsg:
		if time.Now().After(m.errorExpiry) {
			m.lastError = nil
		}
		m.devices = msg
		return m, nil

	case historyMsg:
		if time.Now().After(m.errorExpiry) {
			m.lastError = nil
		}
		// Convert core.HistoryEntry to components.HistoryEntry
		entries := make([]components.HistoryEntry, len(msg))
		for i, h := range msg {
			entries[i] = components.HistoryEntry{
				Track:    h.Track,
				PlayedAt: h.PlayedAt,
			}
		}
		m.history = entries
		return m, nil

	case errMsg:
		m.lastError = msg
		m.errorExpiry = time.Now().Add(5 * time.Second) // Show error for 5 seconds
		return m, nil

	case defaultDeviceSetMsg:
		m.app.defaultDevice = string(msg)
		return m, nil

	case refreshAfterActionMsg:
		return m, tea.Batch(m.fetchState(), m.fetchQueue())

	case searchDebounceMsg:
		if msg.query == m.searchInput.Value() && msg.query != m.lastQuery {
			m.lastQuery = msg.query
			m.searching = true
			return m, m.doSearch(msg.query)
		}

	case searchResultsMsg:
		m.searching = false
		m.searchResults = msg.results
		m.searchErr = msg.err
		m.searchCursor = 0
		return m, nil
	}

	// Forward other messages to textinput when search is active
	if m.showSearch {
		var inputCmd tea.Cmd
		m.searchInput, inputCmd = m.searchInput.Update(msg)
		return m, inputCmd
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (always work)
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	}

	// Help overlay
	if m.showHelp {
		switch msg.String() {
		case "?", "esc":
			m.showHelp = false
		}
		return m, nil
	}

	// Search overlay
	if m.showSearch {
		return m.handleSearchKeyPress(msg)
	}

	// Normal mode
	switch msg.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "?":
		m.showHelp = true
		return m, nil

	case "/":
		m.showSearch = true
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		m.searchResults = nil
		m.searchCursor = 0
		m.searchType = SearchAll
		m.lastQuery = ""
		m.searchErr = nil
		return m, textinput.Blink

	case "esc":
		// Nothing to close in normal mode
		return m, nil

	case "tab":
		m.focusedPanel = (m.focusedPanel + 1) % 4
		return m, nil

	case "shift+tab":
		m.focusedPanel = (m.focusedPanel + 3) % 4
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
		case "d":
			return m, m.setDefaultDevice()
		}
	}

	return m, nil
}

func (m Model) handleSearchKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		m.showSearch = false
		m.searchInput.Blur()
		return m, nil

	case "enter":
		if len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults) {
			result := m.searchResults[m.searchCursor]
			m.showSearch = false
			m.searchInput.Blur()
			return m, m.playSearchResult(result)
		}
		return m, nil

	case "up", "ctrl+p":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
		return m, nil

	case "down", "ctrl+n":
		if m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
		}
		return m, nil

	case "ctrl+t":
		// Cycle through search types
		m.searchType = (m.searchType + 1) % 5
		if m.searchInput.Value() != "" {
			m.searching = true
			return m, m.doSearch(m.searchInput.Value())
		}
		return m, nil

	case "ctrl+q":
		// Add to queue (tracks only)
		if len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults) {
			result := m.searchResults[m.searchCursor]
			if result.Type == SearchTracks {
				m.showSearch = false
				m.searchInput.Blur()
				return m, m.queueSearchResult(result)
			}
		}
		return m, nil
	}

	// Handle text input
	var inputCmd tea.Cmd
	m.searchInput, inputCmd = m.searchInput.Update(msg)
	cmds = append(cmds, inputCmd)

	// Debounce search
	if m.searchInput.Value() != m.lastQuery {
		cmds = append(cmds, tea.Tick(searchDebounce, func(time.Time) tea.Msg {
			return searchDebounceMsg{query: m.searchInput.Value()}
		}))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) togglePlayPause() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if m.state != nil && m.state.IsPlaying {
			_ = m.app.player.Pause(ctx)
		} else {
			_ = m.app.player.Play(ctx)
		}
		return nil
	}
}

type refreshAfterActionMsg struct{}

func (m Model) nextTrack() tea.Cmd {
	return func() tea.Msg {
		_ = m.app.player.Next(context.Background())
		// Small delay to let Spotify update state
		time.Sleep(200 * time.Millisecond)
		return refreshAfterActionMsg{}
	}
}

func (m Model) prevTrack() tea.Cmd {
	return func() tea.Msg {
		_ = m.app.player.Prev(context.Background())
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
			_ = m.app.player.Volume(context.Background(), newVol)
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
			_ = m.app.player.Volume(context.Background(), newVol)
		}
		return nil
	}
}

func (m Model) transferToDevice() tea.Cmd {
	return func() tea.Msg {
		selected := m.devicesView.Selected()
		if selected >= 0 && selected < len(m.devices) {
			device := m.devices[selected]
			_ = m.app.player.TransferPlayback(context.Background(), device.ID, true)
		}
		return nil
	}
}

func (m Model) setDefaultDevice() tea.Cmd {
	return func() tea.Msg {
		selected := m.devicesView.Selected()
		if selected < 0 || selected >= len(m.devices) {
			return nil
		}
		device := m.devices[selected]

		// Save to config file
		if err := saveDefaultDevice(device.Name); err != nil {
			return errMsg(err)
		}
		return defaultDeviceSetMsg(device.Name)
	}
}

// saveDefaultDevice persists the default device name to the config file
func saveDefaultDevice(deviceName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	configPath := filepath.Join(home, ".riffrc")

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create minimal config with just the device
			data = []byte{}
		} else {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	// Parse config
	var rawConfig map[string]interface{}
	if len(data) > 0 {
		if _, err := toml.Decode(string(data), &rawConfig); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
	} else {
		rawConfig = make(map[string]interface{})
	}

	// Get or create defaults section
	defaults, ok := rawConfig["defaults"].(map[string]interface{})
	if !ok {
		defaults = make(map[string]interface{})
		rawConfig["defaults"] = defaults
	}
	defaults["device"] = deviceName

	// Write back
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	defer f.Close()

	_, _ = fmt.Fprintln(f, "# Riff Configuration")
	_, _ = fmt.Fprintln(f, "# https://github.com/tessro/riff")
	_, _ = fmt.Fprintln(f, "")

	encoder := toml.NewEncoder(f)
	encoder.Indent = "  "
	return encoder.Encode(rawConfig)
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
	devicesView := m.devicesView.Render(m.devices, rightWidth-2, topHeight-2, m.focusedPanel == PanelDevices, m.app.defaultDevice)
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
	title := "Riff UI - Keyboard Shortcuts"
	divider := styles.Repeat("═", len(title))

	help := `
  ` + title + `
  ` + divider + `

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
  d            Set as default (★)

  Press ? or Esc to close
`

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(styles.BorderStyle.Render(help))
}

func (m Model) renderSearch() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	b.WriteString(titleStyle.Render("Search"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	// Type filter tabs
	tabs := []string{"All", "Tracks", "Albums", "Artists", "Playlists"}
	activeTabStyle := lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0"))
	tabStyle := lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("243"))
	for i, tab := range tabs {
		if SearchType(i) == m.searchType {
			b.WriteString(activeTabStyle.Render(tab))
		} else {
			b.WriteString(tabStyle.Render(tab))
		}
	}
	b.WriteString("\n\n")

	// Results
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	if m.searchErr != nil {
		b.WriteString(errorStyle.Render("Error: " + m.searchErr.Error()))
	} else if m.searching {
		b.WriteString(subtitleStyle.Render("Searching..."))
	} else if len(m.searchResults) == 0 && m.searchInput.Value() != "" && m.lastQuery != "" {
		b.WriteString(subtitleStyle.Render("No results found"))
	} else {
		maxResults := 10
		for i, result := range m.searchResults {
			if i >= maxResults {
				b.WriteString(subtitleStyle.Render("  ...and more"))
				break
			}

			line := result.Title
			if result.Subtitle != "" {
				line += " " + subtitleStyle.Render(result.Subtitle)
			}

			if i == m.searchCursor {
				b.WriteString(selectedStyle.Render("> " + line))
			} else {
				b.WriteString("  " + line)
			}
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Ctrl+t:filter  ↑/↓:nav  Enter:play  Ctrl+q:queue  Esc:close"))

	content := lipgloss.NewStyle().
		Width(60).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(styles.FocusedBorder.Render(content))
}

// Run starts the TUI application
func Run(clientID string, refreshRate time.Duration, defaultDevice string) error {
	app, err := NewApp(clientID, refreshRate, defaultDevice)
	if err != nil {
		return err
	}

	model := NewModel(app)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err = p.Run()
	return err
}
