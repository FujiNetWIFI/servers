package main

import (
	"slices"
	"sync"
)

type Player struct {
	Name   string // PrintableAscici, no spaces
	Ip     string // www.xxx.yyy.zzz
	Column int    // [0, 255]
	Alive  bool   // is player alive (in game) or dead (watch only)
}

func makePlayer(name string, ip string, column int) Player {
	return Player{
		Name:   name,
		Ip:     ip,
		Column: column,
		Alive:  true,
	}
}

type Game struct {
	sync.Mutex `json:"-"`
	Name       string // PrintableAscici, len < 32 chars
	ServerUrl  string
	Ground     [256]int // this is the ground line. It contains heights [0, 191]
	Players    []Player // Alive and dead players
	CurPlayer  int      `json:"-"` // player that needs to submit the shoot
}

func makeGame(name string, baseurl string) *Game {
	g := &Game{
		ServerUrl: baseurl,
		CurPlayer: -1,
		Name:      name}

	for i := 0; i < len(g.Ground); i++ {
		g.Ground[i] = 20
	}

	return g
}

func (g *Game) M() Map {
	return Map{
		"Name":    g.Name,
		"Players": g.Players,
	}
}

func (g *Game) Add(p Player) {
	g.Lock()
	defer g.Unlock()

	if g.CurPlayer == -1 {
		g.CurPlayer = 0
	}

	g.Players = append(g.Players, p)
}

func (g *Game) Remove(player_name string) {
	g.Lock()
	defer g.Unlock()

	g.Players = slices.DeleteFunc(g.Players, func(each Player) bool { return player_name == each.Name })
}

func (g *Game) SetPlayerActive(p Player) {
	g.Lock()
	defer g.Unlock()

	pos := slices.IndexFunc(g.Players, func(each Player) bool { return p.Name == each.Name })

	g.Players[pos].Alive = true
}

func (g *Game) SetPlayerInActive(p Player) {
	g.Lock()
	defer g.Unlock()

	pos := slices.IndexFunc(g.Players, func(each Player) bool { return p.Name == each.Name })

	g.Players[pos].Alive = false
}

func (g *Game) NextPlayer() (p Player) {
	g.Lock()
	defer g.Unlock()

	if len(g.Players) == 0 {
		return p
	}

	g.CurPlayer = (g.CurPlayer + 1) % len(g.Players)

	return g.Players[g.CurPlayer]
}

// https://gafferongames.com/post/integration_basics/
func (g *Game) Shoot(p Player, angle float32, power int) {
	g.Lock()
	defer g.Unlock()

	// + update ground
	// + update tanks
}

func (g *Game) UpdateLobby() bool {
	return UpdateLobby(5, 0, true, g.Name, g.ServerUrl)
}

func (g *Game) DeleteLobby() bool {
	return RemoveLobby(g.ServerUrl)
}
