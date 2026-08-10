// A terminal Fujirkle client for play testing the server on Linux.
//
// This is a development tool, not a model for the 8 bit clients - those talk to
// the same endpoints through fujinet-lib and parse the packed binary form.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	MOVE_ROLL = 1
	MOVE_BANK = 2

	ROUND_LOBBY    = 0
	ROUND_GAMEOVER = 99

	READY_VIEWING = -2
	READY_YES     = 1
)

type Player struct {
	Name  string `json:"n"`
	Alias int    `json:"a"`
	Score int    `json:"s"`
	Ready int    `json:"y"`
}

type GameState struct {
	Name         string   `json:"n"`
	Prompt       string   `json:"p"`
	Round        int      `json:"r"`
	ActivePlayer int      `json:"a"`
	MoveTime     int      `json:"m"`
	Viewing      int      `json:"v"`
	Dice         string   `json:"d"`
	KeepRoll     string   `json:"k"`
	TurnScore    int      `json:"t"`
	KeptDice     string   `json:"e"`
	Selectable   string   `json:"x"`
	ValidMoves   int      `json:"c"`
	Players      []Player `json:"pl"`
}

type GameTable struct {
	Table      string `json:"t"`
	Name       string `json:"n"`
	CurPlayers int    `json:"p"`
	MaxPlayers int    `json:"m"`
}

var (
	host   = flag.String("host", "http://localhost:8080", "server base url")
	table  = flag.String("table", "dev1", "table to join")
	player = flag.String("player", "", "player name (required unless -list)")
	auto   = flag.Bool("auto", false, "play automatically, for unattended runs")
	games  = flag.Int("games", 0, "exit after this many finished games (0 = never)")
	list   = flag.Bool("list", false, "list tables and exit")
	dev    = flag.Bool("dev", true, "list developer tables rather than public ones")
)

func main() {
	flag.Parse()

	if *list {
		listTables()
		return
	}

	if *player == "" {
		fmt.Fprintln(os.Stderr, "-player is required")
		flag.Usage()
		os.Exit(2)
	}

	fmt.Printf("Connecting to %s as %q on table %q\n", *host, *player, *table)

	input := make(chan string, 1)
	if !*auto {
		go readLines(input)
		printHelp()
	}

	lastRender := ""
	prevRound := ROUND_LOBBY
	finished := 0

	for {
		state, err := fetchState("/state")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if state == nil {
			fmt.Fprintf(os.Stderr, "no such table %q - try -list\n", *table)
			os.Exit(1)
		}

		if view := render(state); view != lastRender {
			fmt.Print(view)
			lastRender = view
		}

		if state.Round == ROUND_GAMEOVER && prevRound != ROUND_GAMEOVER {
			finished++
			if *games > 0 && finished >= *games {
				fmt.Printf("\nfinished %d game(s)\n", finished)
				return
			}
		}
		prevRound = state.Round

		if *auto {
			if autoMove(state) {
				continue
			}
			time.Sleep(400 * time.Millisecond)
			continue
		}

		select {
		case line := <-input:
			if !handle(strings.TrimSpace(line), state) {
				return
			}
			lastRender = ""
		case <-time.After(700 * time.Millisecond):
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
// Server calls
//////////////////////////////////////////////////////////////////////////////

func call(path string) ([]byte, error) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	full := fmt.Sprintf("%s%s%splayer=%s&table=%s",
		strings.TrimRight(*host, "/"), path, sep,
		url.QueryEscape(*player), url.QueryEscape(*table))

	resp, err := http.Get(full)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s", path, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func fetchState(path string) (*GameState, error) {
	body, err := call(path)
	if err != nil {
		return nil, err
	}

	var state GameState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, fmt.Errorf("could not parse state: %v", err)
	}
	if state.Name == "" && state.Players == nil {
		return nil, nil
	}
	return &state, nil
}

func listTables() {
	suffix := ""
	if *dev {
		suffix = "?dev=1"
	}

	resp, err := http.Get(strings.TrimRight(*host, "/") + "/tables" + suffix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tables []GameTable
	if err := json.Unmarshal(body, &tables); err != nil {
		fmt.Fprintf(os.Stderr, "could not parse tables: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-8s %-22s %s\n", "TABLE", "NAME", "PLAYERS")
	for _, t := range tables {
		fmt.Printf("%-8s %-22s %d / %d\n", t.Table, t.Name, t.CurPlayers, t.MaxPlayers)
	}
}

//////////////////////////////////////////////////////////////////////////////
// Rendering
//////////////////////////////////////////////////////////////////////////////

func render(state *GameState) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s\n", strings.Repeat("=", 52))
	fmt.Fprintf(&b, "%s", state.Name)
	if state.Round > ROUND_LOBBY && state.Round < ROUND_GAMEOVER {
		fmt.Fprintf(&b, "   (round %d)", state.Round)
	}
	fmt.Fprintf(&b, "\n%s\n", strings.Repeat("=", 52))

	fmt.Fprintf(&b, "%s", state.Prompt)
	if state.MoveTime > 0 {
		fmt.Fprintf(&b, "   [%ds]", state.MoveTime)
	}
	b.WriteString("\n\n")

	for i, p := range state.Players {
		marker := "  "
		if i == state.ActivePlayer && state.Round > ROUND_LOBBY {
			marker = "> "
		}

		status := ""
		if state.Round == ROUND_LOBBY {
			if p.Ready == READY_YES {
				status = "  READY"
			} else {
				status = "  ..."
			}
		} else if p.Ready == READY_VIEWING {
			status = "  (watching)"
		}

		fmt.Fprintf(&b, "%s%-12s %6d%s\n", marker, p.Name, p.Score, status)
	}

	if state.Round > ROUND_LOBBY && state.Round < ROUND_GAMEOVER {
		b.WriteString("\n")

		if state.KeptDice != "" {
			fmt.Fprintf(&b, "  set aside: %s\n", spaced(state.KeptDice))
		}
		fmt.Fprintf(&b, "  turn score: %d\n\n", state.TurnScore)

		if state.Dice != "" {
			fmt.Fprintf(&b, "  dice:      %s\n", spaced(state.Dice))
			fmt.Fprintf(&b, "  scoring:   %s\n", scoringMarkers(state.Selectable))
			fmt.Fprintf(&b, "  position:  %s\n", positions(len(state.Dice)))
		}
	}

	if state.ValidMoves != 0 {
		b.WriteString("\nYour move. ")
		b.WriteString("Enter dice positions then r or b (e.g. 14b), or ar / ab for all.\n")
	}

	b.WriteString("> ")
	return b.String()
}

func spaced(s string) string {
	parts := make([]string, len(s))
	for i := 0; i < len(s); i++ {
		parts[i] = string(s[i])
	}
	return strings.Join(parts, "  ")
}

func scoringMarkers(mask string) string {
	parts := make([]string, len(mask))
	for i := 0; i < len(mask); i++ {
		if mask[i] == '1' {
			parts[i] = "^"
		} else {
			parts[i] = "."
		}
	}
	return strings.Join(parts, "  ")
}

func positions(n int) string {
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = fmt.Sprintf("%d", i+1)
	}
	return strings.Join(parts, "  ")
}

//////////////////////////////////////////////////////////////////////////////
// Input
//////////////////////////////////////////////////////////////////////////////

func readLines(out chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		out <- scanner.Text()
	}
	close(out)
}

func printHelp() {
	fmt.Println(`
Commands:
  y            ready up / un-ready in the lobby
  <pos>r       keep those dice and roll again   (e.g. 14r)
  <pos>b       keep those dice and bank         (e.g. 135b)
  ar / ab      keep every scoring die, then roll / bank
  t            list tables
  ?            this help
  q            leave and quit`)
}

// Returns false when the client should exit
func handle(line string, state *GameState) bool {
	if line == "" {
		return true
	}

	switch line {
	case "q":
		call("/leave")
		fmt.Println("bye")
		return false
	case "?":
		printHelp()
		return true
	case "t":
		listTables()
		return true
	case "y":
		if _, err := call("/ready"); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		return true
	}

	if state.ValidMoves == 0 {
		fmt.Println("not your turn")
		return true
	}

	action := line[len(line)-1]
	if action != 'r' && action != 'b' {
		fmt.Println("commands end in r (roll) or b (bank) - try ? for help")
		return true
	}

	digits := line[:len(line)-1]

	var mask string
	if digits == "a" {
		mask = state.Selectable
		if !strings.Contains(mask, "1") {
			fmt.Println("nothing in this roll scores")
			return true
		}
	} else {
		var ok bool
		mask, ok = buildMask(digits, len(state.Dice))
		if !ok {
			fmt.Printf("pick positions between 1 and %d\n", len(state.Dice))
			return true
		}
	}

	endpoint := "/roll/"
	if action == 'b' {
		endpoint = "/bank/"
	}

	before := state.TurnScore
	newState, err := fetchState(endpoint + mask)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return true
	}

	// The server silently ignores an illegal set-aside, so tell the player
	if newState != nil && newState.ValidMoves != 0 &&
		newState.TurnScore == before && newState.Dice == state.Dice {
		fmt.Println("that is not a legal set-aside - every die you keep must score")
	}

	return true
}

func buildMask(digits string, size int) (string, bool) {
	if digits == "" || size == 0 {
		return "", false
	}

	mask := []byte(strings.Repeat("0", size))
	for i := 0; i < len(digits); i++ {
		pos := int(digits[i] - '1')
		if pos < 0 || pos >= size {
			return "", false
		}
		mask[pos] = '1'
	}
	return string(mask), true
}

//////////////////////////////////////////////////////////////////////////////
// Unattended play
//////////////////////////////////////////////////////////////////////////////

// Returns true if a request was made, meaning the caller should re-poll at once
func autoMove(state *GameState) bool {
	if state.Round == ROUND_LOBBY {
		me := myPlayer(state)
		if me != nil && me.Ready != READY_YES {
			call("/ready")
			return true
		}
		return false
	}

	if state.ValidMoves == 0 {
		return false
	}

	mask := state.Selectable
	if !strings.Contains(mask, "1") {
		return false
	}

	// Keep everything that scores. The value of that selection is not known
	// until the server applies it, so push or hold on how many dice would be
	// left to re-roll rather than on the turn score so far.
	remaining := 0
	for i := 0; i < len(mask); i++ {
		if mask[i] == '0' {
			remaining++
		}
	}

	push := remaining == 0 || // emptying the pool earns a fresh six
		(remaining >= 3 && state.TurnScore < 1000)

	if push {
		fetchState("/roll/" + mask)
	} else {
		fetchState("/bank/" + mask)
	}
	return true
}

func myPlayer(state *GameState) *Player {
	for i := range state.Players {
		if strings.EqualFold(state.Players[i].Name, *player) {
			return &state.Players[i]
		}
	}
	// The server puts the calling player first
	if len(state.Players) > 0 && state.Viewing == 0 {
		return &state.Players[0]
	}
	return nil
}
