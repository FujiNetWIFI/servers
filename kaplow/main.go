package main

import (
	"log"
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
	TIME      uint64
	RAND      = makeNumGen()
)

const (
	VERSION   = "0.0.5"
	STRINGVER = "kaplow game server  " + VERSION + "/" + runtime.GOOS + " (c) Roger Sen 2023"
)

func main() {

	init_game(
		makeGame("Kaplow!!", "Basic rules (10 sec turn)"),
		makeGame("Kaplow!!", "All shooting crazy (10 sec turn)"))

	init_os_signal()

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

	log.Println("Serving in :8080")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatalf("There's an error with the server, %s", err)
	}
}

func init_scheduler() {
	SCHEDULER := tasks.New()

	TIME = 0

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
	}

}

// return how long has the server been runing
func uptime(start time.Time) string {
	return time.Since(start).String()
}
