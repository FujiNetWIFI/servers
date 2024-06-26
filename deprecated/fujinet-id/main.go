package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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
	DATABASE     *idDB                     // init by Must_init_db
	STARTEDON    = time.Now()              // when the program was started
	SERVERKEY    string                    // read from .env
	RELEASE_MODE = gin.Mode() == "release" // to be used by contract functions.
)

const (
	VERSION   = "0.2.0"
	STRINGVER = "fujinet FujinetId  " + VERSION + "/" + runtime.GOOS + " (c) Roger Sen 2024"
)

//go:embed doc.html
var DOCHTML []byte // init by go:embed & init_html()

func main() {

	var srvaddr string
	var help, version bool

	flag.StringVar(&srvaddr, "srvaddr", ":443", "<address:port> for https server")

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

	init_logger()

	if err := init_server_key(); err != nil {
		ERROR.Fatalf("init_server_key: %s", err)
	}
	init_os_signal()
	Must_init_db()
	init_html(srvaddr)

	router := gin.Default()

	router.GET("/docs", ShowDocs)
	router.POST("/genPubKey", GenPubKey) // username#privatekey --> username!publickey, token
	router.POST("/getPubKey", GetPubKey) // token --> username!publickey, token
	router.GET("/version", ShowStatus)
	router.GET("/license", ShowLicense)

	// https://gist.github.com/denji/12b3a568f092ab951456
	if err := router.RunTLS(srvaddr, "server.crt", "server.key"); err != nil {
		ERROR.Fatalf("Unable to start TLS server (%s)", err)
	}
}

/*
 * Subsystems start here.
 */

// read .env and set SERVERKEY as global variable.
// generate random data with:
// LC_ALL=C tr -dc '[:graph:]' < /dev/urandom | fold -w 64 | head -n1
func init_server_key() error {
	data, err := os.ReadFile(".env")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {

		field, data := split2(line, "=")

		if trim(field) == "SERVERKEY" {
			data = trim(data)

			if len(data) < 64 {
				return fmt.Errorf("SERVERKEY data must be longer than 64 bytes, currently is %d", len(data))
			}

			SERVERKEY = data

			return nil
		}
	}

	return fmt.Errorf("SERVERKEY not found, please generate .env file with SERVERKEY=<data>")
}

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
