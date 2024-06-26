package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/madflojo/tasks"
)

var ( // GLOBALS
	GAMES     = NewConcurrentGameSlice()
	SCHEDULER *tasks.Scheduler
	STARTEDON time.Time = time.Now()
	TIME      uint64    = 0
	RAND                = makeNumGen()
)

const (
	VERSION   = "0.0.5"
	STRINGVER = "kaplow game server  " + VERSION + "/" + runtime.GOOS + " (c) Roger Sen 2023"
)

func main() {

	var srvaddr string
	var help, version bool

	flag.StringVar(&srvaddr, "srvaddr", ":8080", "<address:port> for http server")

	flag.BoolVar(&version, "version", false, "show current version")
	flag.BoolVar(&help, "help", false, "show this help")

	flag.Parse()

	if version {
		fmt.Fprintln(os.Stderr, VERSION)
		return
	}

	if help || len(srvaddr) == 0 {
		flag.PrintDefaults()
		return
	}

	srvaddr = CleanServerAddr(srvaddr)

	init_game(
		makeGame("Kaplow!! (basic 10s)", "http://"+srvaddr),
		makeGame("Kaplow!! (crazy 10s)", "http://"+srvaddr))

	init_os_signal()
	init_scheduler()

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/favicon.ico", handleFavicon)
	router.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets/"))))
	router.Handle("/css/*", http.StripPrefix("/css/", http.FileServer(http.Dir("./assets/css/"))))
	router.Handle("/img/*", http.StripPrefix("/img/", http.FileServer(http.Dir("./assets/img/"))))
	router.Handle("/vendor/*", http.StripPrefix("/vendor/", http.FileServer(http.Dir("./assets/vendor/"))))
	router.Handle("/js/*", http.StripPrefix("/js/", http.FileServer(http.Dir("./assets/js/"))))

	// {id:^[1-9][0-9]*} matches any integer > 0 (and not 0)
	router.Get("/games/{id:^[1-9][0-9]*}/state", GetGameState)     // View the current state as json.
	router.Post("/games/{id:^[1-9][0-9]*/move", PostShoot)         // Apply your player's move and return updated state as json.
	router.Post("/games/{id:^[1-9][0-9]*}/leave", PostLeavePlayer) // For player to leave the game.
	router.Post("/games/{id:^[1-9][0-9]*}/play", PostNewPlayer)    // Add player to the game.

	router.Get("/version", ShowStatus)
	router.Get("/games/", ShowGames) // List all available games in html/json
	router.Get("/", Root)

	slog.Info("Serving in " + srvaddr)
	err := http.ListenAndServe(srvaddr, router)
	if err != nil {
		slog.Error("start", "There's an error with the server: ", err)
		os.Exit(1)
	}
}

func init_scheduler() {
	SCHEDULER := tasks.New()

	SCHEDULER.Add(&tasks.Task{
		Interval: time.Duration(1 * time.Second),
		TaskFunc: ticker("the sec ticker is alive"),
	})

}
func ticker(s string) func() error {

	return func() error {
		if TIME%uint64(time.Minute) == 0 {
			slog.Info("ticker", "Closure", s, "TIME", TIME)
		}

		TIME += 1

		return nil
	}
}

func init_os_signal() {

	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl)

	go SignalHandler(sigchnl)
}

func SignalHandler(sigchan chan os.Signal) {

	for {
		signal := <-sigchan

		switch signal {

		case syscall.SIGTERM:
			slog.Warn("Got SIGTERM. Program will terminate cleanly now.")
			os.Exit(143)
		case syscall.SIGINT:
			slog.Warn("Got SIGINT. Program will terminate cleanly now.")
			os.Exit(137)
		}
	}
}

func init_game(games ...*Game) {

	for i := 0; i < len(games); i++ {
		GAMES.Append(games[i])
		if err := games[i].UpdateLobby(); err != nil {
			slog.Error("init_game", "Unable to update lobbyserver at server", LOBBY_ENDPOINT_UPSERT, "with game", games[i].Name)
		}
	}

}

// return how long has the server been runing
func uptime(start time.Time) string {
	return time.Since(start).String()
}
