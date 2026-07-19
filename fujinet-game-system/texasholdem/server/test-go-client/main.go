// Texas Hold'em test client for the FujiNet game server.
//
// This client speaks the exact same HTTP protocol the 8-bit (Atari / Apple II)
// clients will use, per the original server spec:
//
//  1. GET /tables (and /tables?dev=1) to list tables
//  2. GET /state?table=T&player=P     to join (implicit) and poll game state,
//     with &hash=<z> to skip unchanged states
//  3. GET /move/<CODE>?table=T&player=P to act when it is our turn, where CODE
//     is a 2-character move code from the state's vm (valid moves) array
//  4. GET /leave?table=T&player=P     when exiting
//
// State JSON uses the compact single-character keys designed for 8-bit parsing
// (l, r, p, a, m, v, c, vm, pl, z). Hands are card strings like "KSKH" with
// "??" for hidden cards.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Wire protocol (matches server/readme.md)
// ---------------------------------------------------------------------------

type ValidMove struct {
	Move string `json:"m"`
	Name string `json:"n"`
}

type PlayerState struct {
	Name   string `json:"n"`
	Status int    `json:"s"` // 0 waiting, 1 playing, 2 folded, 3 left, 4 all-in
	Bet    int    `json:"b"`
	Move   string `json:"m"`
	Purse  int    `json:"p"`
	Hand   string `json:"h"` // "KSKH", "????" hidden, "??" folded
}

type GameState struct {
	LastResult   string        `json:"l"`
	Round        int           `json:"r"` // 0 waiting, 1-4 streets, 5 game over
	Pot          int           `json:"p"`
	ActivePlayer int           `json:"a"` // client is always 0; -1 between rounds
	MoveTime     int           `json:"m"`
	Viewing      int           `json:"v"`
	Community    string        `json:"c"` // community cards, e.g. "AS5H2D"
	ValidMoves   []ValidMove   `json:"vm"`
	Players      []PlayerState `json:"pl"`
	Hash         string        `json:"z"`
}

type TableInfo struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

// ---------------------------------------------------------------------------
// Server API
// ---------------------------------------------------------------------------

type api struct {
	base   string
	table  string
	player string
	http   *http.Client
	hash   string
}

func (a *api) get(path string, out any) error {
	resp, err := a.http.Get(a.base + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", path, resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

func (a *api) qs() string {
	return fmt.Sprintf("?table=%s&player=%s", url.QueryEscape(a.table), url.QueryEscape(a.player))
}

// fetchState polls /state. Returns (nil, nil) when the state is unchanged
// (server responded "1" to our hash).
func (a *api) fetchState() (*GameState, error) {
	path := "/state" + a.qs()
	if a.hash != "" {
		path += "&hash=" + a.hash
	}
	resp, err := a.http.Get(a.base + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(body)) == `"1"` {
		return nil, nil // unchanged
	}
	state := &GameState{}
	if err := json.Unmarshal(body, state); err != nil {
		return nil, fmt.Errorf("bad state json: %w", err)
	}
	a.hash = state.Hash
	return state, nil
}

func (a *api) sendMove(code string) error {
	a.hash = "" // force a full state refresh after moving
	return a.get("/move/"+url.PathEscape(code)+a.qs(), nil)
}

func (a *api) leave() error {
	return a.get("/leave"+a.qs(), nil)
}

func (a *api) tables() ([]TableInfo, error) {
	real := []TableInfo{}
	if err := a.get("/tables", &real); err != nil {
		return nil, err
	}
	dev := []TableInfo{}
	if err := a.get("/tables?dev=1", &dev); err != nil {
		return nil, err
	}
	return append(real, dev...), nil
}

// ---------------------------------------------------------------------------
// Card rendering
// ---------------------------------------------------------------------------

var suitSymbols = map[byte]string{'C': "♣", 'D': "♦", 'H': "♥", 'S': "♠"}

var redSuit = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
var blackSuit = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
var hiddenCard = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

// renderCards converts a wire hand string ("KSKH", "??7C", "??") to display form
func renderCards(hand string) string {
	out := []string{}
	for i := 0; i+1 < len(hand); i += 2 {
		pair := hand[i : i+2]
		if pair == "??" {
			out = append(out, hiddenCard.Render("🂠"))
			continue
		}
		value, suit := pair[0], pair[1]
		sym, ok := suitSymbols[suit]
		if !ok {
			out = append(out, pair)
			continue
		}
		display := string(value)
		if value == 'T' {
			display = "10"
		}
		style := blackSuit
		if suit == 'D' || suit == 'H' {
			style = redSuit
		}
		out = append(out, style.Render(display+sym))
	}
	return strings.Join(out, " ")
}

func roundName(r int) string {
	switch r {
	case 0:
		return "Waiting"
	case 1:
		return "Pre-flop"
	case 2:
		return "Flop"
	case 3:
		return "Turn"
	case 4:
		return "River"
	case 5:
		return "Hand complete"
	}
	return fmt.Sprintf("Round %d", r)
}

func statusText(s int) string {
	switch s {
	case 0:
		return "waiting"
	case 2:
		return "folded"
	case 3:
		return "left"
	case 4:
		return "ALL-IN"
	}
	return ""
}

// ---------------------------------------------------------------------------
// Bubbletea model
// ---------------------------------------------------------------------------

type screen int

const (
	screenTables screen = iota
	screenGame
)

type model struct {
	api  *api
	auto bool

	screen        screen
	tables        []TableInfo
	selectedTable int

	state        *GameState
	selectedMove int
	logs         []string
	errorMsg     string
	lastResult   string
}

type stateMsg struct {
	state *GameState
	err   error
}

type tablesMsg struct {
	tables []TableInfo
	err    error
}

type tickMsg time.Time

const pollInterval = 300 * time.Millisecond

func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) fetchStateCmd() tea.Cmd {
	return func() tea.Msg {
		state, err := m.api.fetchState()
		return stateMsg{state: state, err: err}
	}
}

func (m model) fetchTablesCmd() tea.Cmd {
	return func() tea.Msg {
		tables, err := m.api.tables()
		return tablesMsg{tables: tables, err: err}
	}
}

func (m model) sendMoveCmd(code string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.sendMove(code); err != nil {
			return stateMsg{err: err}
		}
		state, err := m.api.fetchState()
		return stateMsg{state: state, err: err}
	}
}

func (m model) Init() tea.Cmd {
	if m.screen == screenTables {
		return m.fetchTablesCmd()
	}
	return tea.Batch(m.fetchStateCmd(), tick())
}

func (m *model) addLog(entry string) {
	m.logs = append(m.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), entry))
	if len(m.logs) > 8 {
		m.logs = m.logs[1:]
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tablesMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Could not list tables: %v", msg.err)
			return m, nil
		}
		m.errorMsg = ""
		m.tables = msg.tables
		return m, nil

	case tickMsg:
		if m.screen != screenGame {
			return m, tick()
		}
		return m, tea.Batch(m.fetchStateCmd(), tick())

	case stateMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Server error: %v", msg.err)
			return m, nil
		}
		m.errorMsg = ""
		if msg.state == nil {
			return m, nil // unchanged (hash short-circuit)
		}
		prev := m.state
		m.state = msg.state
		if m.selectedMove >= len(m.state.ValidMoves) {
			m.selectedMove = 0
		}

		// Log noteworthy transitions
		if prev == nil || prev.Round != m.state.Round {
			m.addLog(roundName(m.state.Round) + communitySuffix(m.state))
		}
		if m.state.LastResult != "" && m.state.LastResult != m.lastResult {
			m.lastResult = m.state.LastResult
			m.addLog(m.state.LastResult)
		}

		// Auto-play: check/call when possible, otherwise fold
		if m.auto && m.state.ActivePlayer == 0 && len(m.state.ValidMoves) > 0 {
			return m, m.sendMoveCmd(autoPick(m.state.ValidMoves))
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func communitySuffix(s *GameState) string {
	if s.Community == "" {
		return ""
	}
	return " - board: " + s.Community
}

func autoPick(moves []ValidMove) string {
	for _, code := range []string{"CH", "CA"} {
		for _, vm := range moves {
			if vm.Move == code {
				return vm.Move
			}
		}
	}
	return "FO"
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c", "q":
		if m.screen == screenGame {
			m.api.leave()
		}
		return m, tea.Quit
	}

	if m.screen == screenTables {
		switch key {
		case "up", "k":
			if m.selectedTable > 0 {
				m.selectedTable--
			}
		case "down", "j":
			if m.selectedTable < len(m.tables)-1 {
				m.selectedTable++
			}
		case "r":
			return m, m.fetchTablesCmd()
		case "enter":
			if len(m.tables) > 0 {
				m.api.table = m.tables[m.selectedTable].Table
				m.screen = screenGame
				m.addLog("Joined table " + m.api.table + " as " + m.api.player)
				return m, tea.Batch(m.fetchStateCmd(), tick())
			}
		}
		return m, nil
	}

	// Game screen
	myTurn := m.state != nil && m.state.ActivePlayer == 0 && len(m.state.ValidMoves) > 0
	switch key {
	case "up", "k":
		if myTurn && m.selectedMove > 0 {
			m.selectedMove--
		}
	case "down", "j":
		if myTurn && m.selectedMove < len(m.state.ValidMoves)-1 {
			m.selectedMove++
		}
	case "enter":
		if myTurn {
			move := m.state.ValidMoves[m.selectedMove]
			m.addLog("You chose " + move.Name)
			return m, m.sendMoveCmd(move.Move)
		}
	case "a":
		m.auto = !m.auto
		m.addLog(fmt.Sprintf("Auto-play %v", map[bool]string{true: "ON", false: "OFF"}[m.auto]))
		if m.auto && myTurn {
			return m, m.sendMoveCmd(autoPick(m.state.ValidMoves))
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if myTurn {
			idx, _ := strconv.Atoi(key)
			idx--
			if idx >= 0 && idx < len(m.state.ValidMoves) {
				move := m.state.ValidMoves[idx]
				m.addLog("You chose " + move.Name)
				return m, m.sendMoveCmd(move.Move)
			}
		}
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Views
// ---------------------------------------------------------------------------

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	headStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	moveStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	winStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	activeStyle = lipgloss.NewStyle().Bold(true)
)

func (m model) View() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("🃏 Texas Hold'em - FujiNet Test Client"))
	s.WriteString("\n\n")

	if m.errorMsg != "" {
		s.WriteString(errStyle.Render("⚠ " + m.errorMsg))
		s.WriteString("\n\n")
	}

	if m.screen == screenTables {
		m.viewTables(&s)
	} else {
		m.viewGame(&s)
	}

	if len(m.logs) > 0 {
		s.WriteString(dimStyle.Render("Activity:"))
		s.WriteString("\n")
		for _, entry := range m.logs {
			s.WriteString(dimStyle.Render("  " + entry))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	if m.screen == screenTables {
		s.WriteString(dimStyle.Render("↑/↓ select · enter join · r refresh · q quit"))
	} else {
		auto := ""
		if m.auto {
			auto = " · AUTO-PLAY ON"
		}
		s.WriteString(dimStyle.Render("↑/↓ or 1-9 select move · enter confirm · a auto-play · q leave & quit" + auto))
	}
	s.WriteString("\n")
	return s.String()
}

func (m model) viewTables(s *strings.Builder) {
	s.WriteString(headStyle.Render("Choose a table on " + m.api.base))
	s.WriteString("\n\n")
	if len(m.tables) == 0 {
		s.WriteString("  (no tables found - is the server running?)\n\n")
		return
	}
	for i, t := range m.tables {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.selectedTable {
			cursor = "▶ "
			style = activeStyle
		}
		s.WriteString(style.Render(fmt.Sprintf("%s%-12s %s (%d/%d players)", cursor, t.Table, t.Name, t.CurPlayers, t.MaxPlayers)))
		s.WriteString("\n")
	}
	s.WriteString("\n")
}

func (m model) viewGame(s *strings.Builder) {
	if m.state == nil {
		s.WriteString("Connecting to table " + m.api.table + "...\n\n")
		return
	}
	st := m.state

	header := fmt.Sprintf("%s | Pot: $%d", roundName(st.Round), st.Pot)
	if st.Viewing == 1 {
		header += " | SPECTATING"
	}
	if st.MoveTime > 0 {
		header += fmt.Sprintf(" | ⏱ %ds", st.MoveTime)
	}
	s.WriteString(headStyle.Render(header))
	s.WriteString("\n\n")

	if st.Community != "" {
		s.WriteString("  Board: " + renderCards(st.Community))
		s.WriteString("\n\n")
	}

	for i, p := range st.Players {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == st.ActivePlayer {
			cursor = "▶ "
			style = activeStyle
		}
		name := p.Name
		if i == 0 {
			name += " (You)"
		}
		line := fmt.Sprintf("%s%-16s $%-5d", cursor, name, p.Purse)
		if p.Bet > 0 {
			line += fmt.Sprintf(" bet:$%-4d", p.Bet)
		} else {
			line += "          "
		}
		if extra := statusText(p.Status); extra != "" {
			line += " [" + extra + "]"
		}
		if p.Move != "" {
			line += " " + p.Move
		}
		s.WriteString(style.Render(line))
		if p.Hand != "" {
			s.WriteString("  " + renderCards(p.Hand))
		}
		s.WriteString("\n")
	}
	s.WriteString("\n")

	if st.Round == 5 && st.LastResult != "" {
		s.WriteString(winStyle.Render("🏆 " + st.LastResult))
		s.WriteString("\n\n")
	} else if st.Round == 0 && st.LastResult != "" {
		s.WriteString(dimStyle.Render(st.LastResult))
		s.WriteString("\n\n")
	}

	if st.ActivePlayer == 0 && len(st.ValidMoves) > 0 && st.Viewing == 0 {
		s.WriteString(moveStyle.Render("Your turn:"))
		s.WriteString("\n")
		for i, vm := range st.ValidMoves {
			cursor := "   "
			style := lipgloss.NewStyle()
			if i == m.selectedMove {
				cursor = " ▶ "
				style = activeStyle
			}
			s.WriteString(style.Render(fmt.Sprintf("%s%d. %s", cursor, i+1, vm.Name)))
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}
}

// ---------------------------------------------------------------------------
// Headless mode: play like an 8-bit client would, printing events to stdout.
// Useful for scripted end-to-end verification of the client protocol.
// ---------------------------------------------------------------------------

func runHeadless(a *api, hands int, timeout time.Duration) error {
	fmt.Printf("Headless: joining table %q as %q on %s\n", a.table, a.player, a.base)
	deadline := time.Now().Add(timeout)
	handsSeen := 0
	lastRound := -1
	lastResult := ""

	for time.Now().Before(deadline) {
		state, err := a.fetchState()
		if err != nil {
			return err
		}
		if state == nil { // unchanged
			time.Sleep(pollInterval)
			continue
		}

		if state.Round != lastRound {
			lastRound = state.Round
			line := fmt.Sprintf("== %s | pot $%d", roundName(state.Round), state.Pot)
			if state.Community != "" {
				line += " | board " + state.Community
			}
			if len(state.Players) > 0 && state.Players[0].Hand != "" {
				line += " | hand " + state.Players[0].Hand
			}
			fmt.Println(line)
		}
		if state.LastResult != "" && state.LastResult != lastResult {
			lastResult = state.LastResult
			fmt.Println("** " + state.LastResult)
			if state.Round == 5 {
				handsSeen++
				if handsSeen >= hands {
					a.leave()
					fmt.Printf("Done: %d hand(s) completed.\n", handsSeen)
					return nil
				}
			}
		}

		if state.ActivePlayer == 0 && len(state.ValidMoves) > 0 && state.Viewing == 0 {
			code := autoPick(state.ValidMoves)
			fmt.Printf("-> playing %s\n", code)
			if err := a.sendMove(code); err != nil {
				return err
			}
			continue
		}

		time.Sleep(pollInterval)
	}
	a.leave()
	return fmt.Errorf("timed out after %v with %d/%d hands completed", timeout, handsSeen, hands)
}

// ---------------------------------------------------------------------------

// https://th.carr-designs.com/tables


func main() {
	server := flag.String("server", "https://th.carr-designs.com", "Game server base URL")
	table := flag.String("table", "", "Table to join (empty = interactive table picker)")
	name := flag.String("name", "Human", "Player name")
	auto := flag.Bool("auto", false, "Start with auto-play enabled")
	headless := flag.Bool("headless", false, "No UI: auto-play and print events (implies -auto)")
	hands := flag.Int("hands", 1, "Headless: number of hands to play before exiting")
	timeout := flag.Duration("timeout", 5*time.Minute, "Headless: maximum run time")
	flag.Parse()

	a := &api{
		base:   strings.TrimRight(*server, "/"),
		table:  *table,
		player: *name,
		http:   &http.Client{Timeout: 10 * time.Second},
	}

	if *headless {
		if a.table == "" {
			a.table = "dev3"
		}
		if err := runHeadless(a, *hands, *timeout); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	m := model{api: a, auto: *auto}
	if a.table == "" {
		m.screen = screenTables
	} else {
		m.screen = screenGame
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running client:", err)
		os.Exit(1)
	}
}
