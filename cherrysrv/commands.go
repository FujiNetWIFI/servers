package main

import (
	"runtime"
)

func init_commands() {
	COMMANDS["login"] = do_login
	COMMANDS["logoff"] = do_logoff
	COMMANDS["who"] = do_who
	COMMANDS["users"] = do_users
	COMMANDS["nusers"] = do_nusers
	COMMANDS["say"] = do_say
	COMMANDS["clock"] = do_clock
	COMMANDS["help"] = do_help

}

func do_help(clt *Client, args string) {

	clt.SayN(">/help>", []string{"/login <email> - login to cherry server",
		"/who                       - show my nickname",
		"/help                      - this command",
		"/users                     - who is logged?",
		"/logoff                    - logoff"})

}

// show internal timer
func do_clock(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/clock>0>/clock requires you to be logged")

		return
	}

	clt.Say(">/clock>0>%d", TIME)
}

// count number of users logged
func do_nusers(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/nusers>0>/nusers requires you to be logged")

		return
	}

	/* Do command */

	NumUsers := 0

	CountUsers := func(key string, v *Client) bool {

		// we don't want to count users that are logging out
		if v.status != USER_LOGGINOUT {
			NumUsers++
		}

		return true
	}

	CLIENTS.Range(CountUsers)

	clt.Say(">/nusers>0>%d", NumUsers)
}

// talk to other logged users
func do_say(clt *Client, args string) {

	clt.SayToAllButMe(args)
}

// update login levels. Unused for now
func sys_log(clt *Client, args string) {

	if no(args) {
		status := []string{INFO.String(),
			WARN.String(), ERROR.String(),
			LOGGER.String(),
			DEBUG.String(), LOGGER.String()}

		clt.SayN(">/log>", status)

		return
	}

	logger, onoff := split2(args, " ")

	// Do command

	err := update_log_level(logger, onoff)

	if err != nil {
		clt.Say(">/log>0>unable to change %s to %s", logger, onoff)
		return
	}

	clt.Say(">/log>0>loglevel updated: %s to %s", logger, onoff)

}

// show logged users
func do_users(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/users>0>/users requires you to be logged")

		return
	}

	/* Do command */

	var out []string

	print_key := func(key string, v *Client) bool {

		if v.status != USER_LOGGINOUT {
			out = append(out, "@"+key)
		}
		return true
	}

	CLIENTS.Range(print_key)

	clt.SayN(">/users>", out)
}

// login user. No password required
func do_login(clt *Client, args string) {

	/* Check params */

	if clt.isLogged() {
		clt.Say(">/login>0>you're already logged in")

		return
	}

	if no(args) {
		clt.Say(">/login>0>/login <account>")

		return
	}

	username, err := ValidUsername(args)

	if err != nil {
		clt.Say(err.Error())
		WARN.Printf(">/login>0>@user %s unable to login due to: %s", args, err.Error())

		return
	}

	/* Do command */

	oldName := clt.name

	clt.name = username
	clt.status = USER_LOGGED
	CLIENTS.Store(clt.name, clt)
	CLIENTS.Delete(oldName)

	/* Update player */

	clt.Say("/login>0>you're now @%s", clt.name)
	clt.BroadcastButMe(">#main>!login>@%s has joined the room", clt.name)

	INFO.Printf("%s has logged in as @%s", oldName, clt.name)
}

// logoff user
func do_logoff(clt *Client, args string) {

	/* Do command */
	clt.status = USER_LOGGINOUT

	clt.Say(">/logoff>0>Goodbye @%s", clt.name)

	clt.BroadcastButMe(">#main>!logoff>@%s is leaving", clt.name)

	INFO.Printf("@%s logged off (%s)", clt.name, clt.conn.RemoteAddr())

	clt.Close()

	runtime.Goexit()
}

// show name of logged user
func do_who(clt *Client, args string) {
	clt.Say(">/who>0>@%s", clt.name)
}
