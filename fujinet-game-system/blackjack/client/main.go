// Terminal client for play testing the server; the 8 bit clients use the
// packed binary form instead
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
	"strconv"
	"strings"
	"time"
)

const (
	ROUND_WAITING = 0
	ROUND_BETTING = 1
	ROUND_PLAYING = 2
	ROUND_DEALER  = 3
	ROUND_OVER    = 5

	STATUS_PLAYING = 1

	FLAG_SOFT      = 1
	FLAG_BLACKJACK = 2
	FLAG_BUST      = 4
	FLAG_DOUBLED   = 8
)

var resultNames = []string{"", "win", "lose", "push", "blackjack", "surrender"}

type ValidMove struct {
	Move string `json:"m"`
	Name string `json:"n"`
}

type Hand struct {
	Bet    int    `json:"b"`
	Total  int    `json:"t"`
	Flags  int    `json:"f"`
	Result int    `json:"r"`
	Cards  string `json:"c"`
}

type Player struct {
	Name   string `json:"n"`
	Status int    `json:"s"`
	Purse  int    `json:"p"`
	Move   string `json:"m"`
	Hands  []Hand `json:"hs"`
}

type GameState struct {
	LastResult   string      `json:"l"`
	Round        int         `json:"r"`
	ActivePlayer int         `json:"a"`
	ActiveHand   int         `json:"ah"`
	MoveTime     int         `json:"m"`
	Viewing      int         `json:"v"`
	Dealer       string      `json:"d"`
	DealerTotal  int         `json:"dt"`
	DealerFlags  int         `json:"df"`
	ValidMoves   []ValidMove `json:"vm"`
	Players      []Player    `json:"pl"`
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
	hands  = flag.Int("hands", 0, "exit after this many finished hands (0 = never)")
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
	prevRound := ROUND_WAITING
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

		if state.Round == ROUND_OVER && prevRound != ROUND_OVER {
			finished++
			if *hands > 0 && finished >= *hands {
				fmt.Printf("\nfinished %d hand(s)\n", finished)
				call("/leave")
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
		case line, ok := <-input:
			if !ok || !handle(strings.TrimSpace(line), state) {
				call("/leave")
				return
			}
			lastRender = ""
		case <-time.After(700 * time.Millisecond):
		}
	}
}

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
	if string(body) == "null" {
		return nil, nil
	}
	var state GameState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, fmt.Errorf("could not parse state: %v", err)
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

func spacedCards(s string) string {
	parts := []string{}
	for i := 0; i+1 < len(s); i += 2 {
		parts = append(parts, s[i:i+2])
	}
	return strings.Join(parts, " ")
}

func describeTotal(total, flags int) string {
	switch {
	case flags&FLAG_BLACKJACK != 0:
		return "blackjack"
	case flags&FLAG_BUST != 0:
		return fmt.Sprintf("%d bust", total)
	case flags&FLAG_SOFT != 0:
		return fmt.Sprintf("soft %d", total)
	}
	return strconv.Itoa(total)
}

func render(state *GameState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", strings.Repeat("=", 56))
	fmt.Fprintf(&b, "%s", state.LastResult)
	if state.MoveTime > 0 {
		fmt.Fprintf(&b, "   [%ds]", state.MoveTime)
	}
	if state.Viewing != 0 {
		b.WriteString("   (watching - waiting for a seat)")
	}
	fmt.Fprintf(&b, "\n%s\n", strings.Repeat("=", 56))

	if state.Dealer != "" {
		fmt.Fprintf(&b, "  Dealer      %-24s %s\n\n", spacedCards(state.Dealer), describeTotal(state.DealerTotal, state.DealerFlags))
	}

	for i, p := range state.Players {
		marker := "  "
		if i == state.ActivePlayer {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%-10s %5d  %s\n", marker, p.Name, p.Purse, p.Move)
		for h, hand := range p.Hands {
			if hand.Cards == "" {
				continue
			}
			handMarker := "    "
			if i == state.ActivePlayer && h == state.ActiveHand && len(p.Hands) > 1 {
				handMarker = "  * "
			}
			extra := ""
			if hand.Flags&FLAG_DOUBLED != 0 {
				extra = " doubled"
			}
			if hand.Result != 0 {
				extra += " - " + resultNames[hand.Result]
			}
			fmt.Fprintf(&b, "%s%-24s %-10s bet %d%s\n", handMarker, spacedCards(hand.Cards), describeTotal(hand.Total, hand.Flags), hand.Bet, extra)
		}
	}

	if len(state.ValidMoves) > 0 {
		b.WriteString("\nYour move: ")
		for i, m := range state.ValidMoves {
			fmt.Fprintf(&b, " %d) %s ", i+1, m.Name)
		}
		b.WriteString("\n")
	}
	b.WriteString("> ")
	return b.String()
}

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
  1-5          pick that move from the menu
  hi st dd sp su rb b1-b4   or send a move code directly
  t            list tables
  ?            this help
  q            leave and quit`)
}

// Returns false when the client should exit
func handle(line string, state *GameState) bool {
	switch line {
	case "":
		return true
	case "q":
		fmt.Println("bye")
		return false
	case "?":
		printHelp()
		return true
	case "t":
		listTables()
		return true
	}

	if len(state.ValidMoves) == 0 {
		fmt.Println("not your turn")
		return true
	}

	code := strings.ToUpper(line)
	if n, err := strconv.Atoi(line); err == nil {
		if n < 1 || n > len(state.ValidMoves) {
			fmt.Printf("pick 1 to %d\n", len(state.ValidMoves))
			return true
		}
		code = state.ValidMoves[n-1].Move
	}
	if _, err := call("/move/" + code); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	return true
}

// Returns true if a request was made, meaning the caller should re-poll at once
func autoMove(state *GameState) bool {
	if len(state.ValidMoves) == 0 {
		return false
	}
	code := state.ValidMoves[0].Move
	if state.Round == ROUND_PLAYING {
		hand := state.Players[0].Hands[state.ActiveHand]
		code = "HI"
		if hand.Total >= 17 || hand.Total >= 12 && state.DealerTotal <= 6 {
			code = "ST"
		}
	}
	call("/move/" + code)
	return true
}
