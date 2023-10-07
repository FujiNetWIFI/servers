package main

import (
	"bytes"
	_ "embed"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madflojo/tasks"
)

var (
	WARN   CustomLogger
	INFO   CustomLogger
	ERROR  CustomLogger
	DEBUG  CustomLogger
	LOGGER CustomLogger
	DB     CustomLogger
)

var (
	DATABASE  *lobbyDB
	SCHEDULER *tasks.Scheduler
	TIME      uint64
	STARTEDON time.Time
)

const (
	VERSION   = "5.0.1"
	STRINGVER = "fujinet persistent lobby  " + VERSION + "/" + runtime.GOOS + " (c) Roger Sen 2023"
)

//go:embed doc.html
var DOCHTML []byte

//go:embed servers.html
var SERVERS_HTML []byte

func main() {

	var srvaddr, evtaddr string
	var help bool

	flag.StringVar(&srvaddr, "srvaddr", ":8080", "<address:port> for http server")
	flag.StringVar(&evtaddr, "evtaddr", "", "<http> for event server webhook")

	flag.BoolVar(&help, "help", false, "show this help")

	flag.Parse()

	if help || len(srvaddr) == 0 {
		flag.PrintDefaults()
		return
	}

	init_logger()
	init_os_signal()
	init_scheduler()
	init_time()
	init_db()
	init_html(srvaddr)

	router := gin.Default()

	router.GET("/", ShowServersHtml)
	router.GET("/docs", ShowDocs)
	router.GET("/viewFull", ShowServers)
	router.GET("/view", ShowServersMinimised)
	router.GET("/version", ShowStatus)
	router.POST("/server", UpsertServer)
	router.DELETE("/server", DeleteServer)

	router.Run(srvaddr)

}

/*
 * Subsystems start here.
 */

func init_logger() {

	INFO = NewCustomLogger("info", "INFO: ", log.LstdFlags)
	WARN = NewCustomLogger("warn", "WARN: ", log.LstdFlags)
	ERROR = NewCustomLogger("error", "ERROR: ", log.LstdFlags)
	LOGGER = NewCustomLogger("logger", "LOGGER: ", log.LstdFlags)
	DEBUG = NewCustomLogger("debug", "DEBUG: ", log.LstdFlags|log.Lshortfile)
	DB = NewCustomLogger("db", "DB: ", log.LstdFlags)

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

// save start of the program time
func init_time() {
	STARTEDON = time.Now()
}

// return how long has the server been runing
func uptime(start time.Time) string {
	return time.Since(start).String()
}

// replace tags on DOCHTML
func init_html(srvaddr string) {

	srvaddr = strings.ToLower(srvaddr)

	if !strings.HasPrefix(srvaddr, "http://") {
		srvaddr = "http://" + srvaddr
	}

	if !strings.HasSuffix(srvaddr, "/") {
		srvaddr = srvaddr + "/"
	}

	DOCHTML = bytes.ReplaceAll(DOCHTML, []byte("$$srvaddr$$"), []byte(srvaddr))
	DOCHTML = bytes.ReplaceAll(DOCHTML, []byte("$$version$$"), []byte(VERSION))
}
