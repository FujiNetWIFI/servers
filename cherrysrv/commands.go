package main

import (
	"fmt"
	"runtime"
	"time"
)

func init_commands() {
	COMMANDS["login"] = do_login
	COMMANDS["logoff"] = do_logoff
	COMMANDS["who"] = do_who
	COMMANDS["users"] = do_users
	COMMANDS["say"] = do_say
	COMMANDS["clock"] = do_clock
	COMMANDS["help"] = do_help

}

func do_help(clt *Client, args string) {

	clt.OKPrintfN([]string{"/login <email> - login to cherry server",
		"/who                       - show my nickname",
		"/help                      - this command",
		"/users                     - who is logged?",
		"/logoff                    - logoff"})

}

func do_clock(clt *Client, args string) {

	if !clt.isLogged() {
		clt.FAILPrintf("/clock requires you to be logged")

		return
	}

	clt.OKPrintf("%d", TIME)
}

func do_say(clt *Client, args string) {

	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("%s : %s : %s\n", clt.name, now, args)

	clt.SayToAllButMe(line)
}

func sys_log(clt *Client, args string) {

	if no(args) {
		status := []string{INFO.String(),
			WARN.String(), ERROR.String(),
			LOGGER.String(), DB.String(),
			DEBUG.String(), LOGGER.String()}

		clt.OKPrintfN(status)

		return
	}

	logger, onoff := split2(args, " ")

	// Do command

	err := update_log_level(logger, onoff)

	if err != nil {
		clt.FAILPrintf("unable to change %s to %s", logger, onoff)
		return
	}

	clt.OKPrintf("loglevel updated: %s to %s", logger, onoff)

}

func do_users(clt *Client, args string) {

	if !clt.isLogged() {
		clt.FAILPrintf("/users requires you to be logged")

		return
	}

	/* Do command */

	var out []string

	print_key := func(key string, v *Client) bool {
		out = append(out, key)
		return true
	}

	CLIENTS.Range(print_key)

	clt.OKPrintfN(out)
}

func do_login(clt *Client, args string) {

	/* Check params */

	if clt.isLogged() {
		clt.FAILPrintf("you're already logged in")

		return
	}

	if no(args) {
		clt.OKPrintf("/login <email>")

		return
	}

	username, err := ValidUsername(args)

	if err != nil {
		clt.FAILPrintf(err.Error())
		WARN.Printf("User %s unable to login due to: %s", args, err.Error())

		return
	}

	/* Do command */

	oldName := clt.name

	clt.name = username
	clt.status = USER_LOGGED
	CLIENTS.Store(clt.name, clt)
	CLIENTS.Delete(oldName)

	/* Update player */

	clt.OKPrintf("you're now %s", clt.name)
	clt.SayToAllButMe(clt.name + " has joined the room")

	INFO.Printf("%s has logged in as %s", oldName, clt.name)
}

func do_logoff(clt *Client, args string) {

	/* Do command */
	clt.status = USER_LOGGINOUT

	clt.OKPrintf("Goodbye %s", clt.name)

	clt.SayToAllButMe(clt.name + "is leaving")

	INFO.Printf("%s logged off (%s)", clt.name, clt.conn.RemoteAddr())

	clt.Close()

	runtime.Goexit()
}

func do_who(clt *Client, args string) {
	clt.OKPrintf(clt.name)
}
