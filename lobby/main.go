package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lrita/cmap"
	"github.com/madflojo/tasks"
)

var (
	WARN   CustomLogger
	INFO   CustomLogger
	ERROR  CustomLogger
	DEBUG  CustomLogger
	LOGGER CustomLogger
)

var (
	GAMESRV   cmap.Map[string, *GameServer] // to store game servers
	SCHEDULER *tasks.Scheduler
	TIME      uint64
)

const (
	VERSION   = "2.0.0"
	STRINGVER = "fujinet lobby " + VERSION + "/" + runtime.GOOS + " (c) Roger Sen 2023"
)

func main() {

	init_logger()
	init_os_signal()
	init_scheduler()

	var srvaddr string
	var help bool

	flag.StringVar(&srvaddr, "srvaddr", ":8080", "<address:port> for http server")
	flag.BoolVar(&help, "help", false, "show this help")

	flag.Parse()

	if help || len(srvaddr) == 0 {
		flag.PrintDefaults()
		return
	}

	router := gin.Default()

	router.GET("/viewFull", ShowServers)
	router.GET("/view", ShowServersMinimised)
	router.POST("/server", UpsertServer)

	router.Run(srvaddr)

}

/*
 *      Subsystems start here.
 */

func init_logger() {

	INFO = NewCustomLogger("info", "INFO: ", log.LstdFlags)
	WARN = NewCustomLogger("warn", "WARN: ", log.LstdFlags)
	ERROR = NewCustomLogger("error", "ERROR: ", log.LstdFlags)
	LOGGER = NewCustomLogger("logger", "LOGGER: ", log.LstdFlags)
	DEBUG = NewCustomLogger("debug", "DEBUG: ", log.LstdFlags|log.Lshortfile)

	value, ok := os.LookupEnv("LOG_LEVEL")

	if ok && value == "PROD" {
		DEBUG.SetActive(false)
	}
}

func init_scheduler() error {
	SCHEDULER := tasks.New()

	TIME = 0

	SCHEDULER.Add(&tasks.Task{
		Interval: time.Duration(1 * time.Second),
		TaskFunc: ticker("a 1 sec ticker"),
	})

	return nil

}

// TODO, we should be able to add parameters to the function to exec w/o closures
func ticker(s string) func() error {

	return func() error {

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
			WARN.Println("Got SIGTERM. Program will terminate cleanly now.")
			os.Exit(143)
		case syscall.SIGINT:
			WARN.Println("Got SIGINT. Program will terminate cleanly now.")
			os.Exit(137)
		}
	}
}
