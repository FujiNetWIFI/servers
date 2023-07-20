/*
   cherrysrv is a chat server for 8bit computers
   Copyright (C) 2023  Roger Sen roger.sen@gmail.com

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/lrita/cmap"
	"github.com/madflojo/tasks"
)

/* https://github.com/avelino/awesome-go */
/* Confing ini files https://github.com/sasbury/mini */
/*
https://github.com/skx/evalfilter
https://github.com/madflojo/tasks
*/

var (
	WARN   CustomLogger
	INFO   CustomLogger
	ERROR  CustomLogger
	DEBUG  CustomLogger
	LOGGER CustomLogger
)

type do_command func(*Client, string)

// This is our world!
var (
	COMMANDS  = make(map[string]do_command)
	CLIENTS   cmap.Map[string, *Client] // CLIENTS  cmap.Cmap
	CHANNELS  cmap.Map[string, *Channel]
	SCHEDULER *tasks.Scheduler
	TIME      uint64
	STARTEDON time.Time
)

const (
	VERSION   = "3.0.1"
	STRINGVER = "cherry srv " + VERSION + "/" + runtime.GOOS + " (c) Roger Sen 2023"
)

func main() {

	var srvaddr string
	var help bool

	flag.StringVar(&srvaddr, "srvaddr", "", "<address:port> for tcp4 server")
	flag.BoolVar(&help, "help", false, "show this help")

	flag.Parse()

	if help || len(srvaddr) == 0 {
		flag.PrintDefaults()
		return
	}

	init_logger()
	init_os_signal()
	init_commands()
	init_scheduler()
	init_time()

	rand.Seed(time.Now().UnixNano())

	TCPAddr, err := net.ResolveTCPAddr("tcp", srvaddr)
	if err != nil {
		ERROR.Fatalf("Unable to resolve address on tcp4://%s (%s)", srvaddr, err)
		return
	}

	server, err := net.ListenTCP("tcp4", TCPAddr)
	if err != nil {
		ERROR.Fatalf("Unable to serve on tcp4://%s (%s)", srvaddr, err)
		return
	}
	defer server.Close()

	INFO.Printf("Started %s", STRINGVER)
	INFO.Printf("Ready to serve on tcp://%s (tcp)", srvaddr)

	// We create tha main channel

	main_channel := NewChannelMain("#main")
	CHANNELS.Store(main_channel.Key(), main_channel)
	DEBUG.Printf("adding %s to CHANNELS", main_channel)

	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			WARN.Printf("Unable to accept connection on localhost:1512 (%s)", err)
			continue
		}
		go newClient(conn).clientLoop()
	}
}

/*
 *	Subsystems start here.
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

func update_log_level(logger string, onoff string) error {

	logger = strings.ToLower(logger)
	onoff = strings.ToLower(onoff)

	if logger == "logger" {
		LOGGER.Printf("unable to change the operation of the logger LOGGER")
		return fmt.Errorf("unable to change the operation of the logger '%s'", logger)
	}

	var newstatus bool

	switch onoff {
	case "on":
		newstatus = true
	case "off":
		newstatus = false
	default:
		LOGGER.Printf("unable to change logger to status '%s'. Only (on/off) are valid.", onoff)
		return fmt.Errorf("unable to change logger to status '%s'. Only (on/off) are valid", onoff)
	}

	switch logger {
	case "info":
		INFO.SetActive(newstatus)
	case "warn":
		WARN.SetActive(newstatus)
	case "error":
		ERROR.SetActive(newstatus)
	case "debug":
		DEBUG.SetActive(newstatus)

	default:
		LOGGER.Printf("unable to update logger. '%s' is not a valid loglevel", logger)
		return fmt.Errorf("unable to update logger. '%s' is not a valid loglevel", logger)
	}

	LOGGER.Printf("logger '%s' updated to status '%s'", logger, onoff)

	return nil
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

// save when program started
func init_time() error {
	STARTEDON = time.Now()

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
	signal.Ignore(syscall.SIGURG, syscall.SIGWINCH) // SIGURG and SIGWINCH pop in macOS. Filter it out
	go SignalHandler(sigchnl)
}

func SignalHandler(sigchan chan os.Signal) {

	for {
		signal := <-sigchan

		switch signal {

		case syscall.SIGTERM:
			WARN.Println("Got SIGTERM. Program will terminate cleanly now.")
			Broadcast(">#main>!shutdown>Shutting down the server, it will re-start in a few minutes")
			os.Exit(143)
		case syscall.SIGINT:
			WARN.Println("Got SIGINT. Program will terminate cleanly now.")
			Broadcast(">#main>!shutdown>Shutting down the server, it will re-start in a few minutes")
			os.Exit(137)
		default:
			INFO.Printf("Received signal %s. No action taken.", signal)
		}
	}
}

func uptime(start time.Time) string {
	return time.Since(start).String()
}
