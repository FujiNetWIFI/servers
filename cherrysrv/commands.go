package main

import (
	"runtime"
	"sort"
	"strings"
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
	COMMANDS["version"] = do_version
	COMMANDS["uptime"] = do_uptime
	COMMANDS["join"] = do_join
	COMMANDS["hjoin"] = do_hjoin
	COMMANDS["leave"] = do_leave
	COMMANDS["list"] = do_list
	COMMANDS["license"] = do_license
	COMMANDS["history"] = do_history
}

func do_help(clt *Client, args string) {

	clt.SayN(">/help>",
		[]string{"/login <nick> - login to cherry server",
			"/who                       - show my nickname",
			"/help                      - this command",
			"/users                     - who is logged?",
			"/users <#channel>          - who is in this channel?",
			"/nusers                    - number of users",
			"/nusers <#channel>         - number of users in channel",
			"/list                      - show available public channels",
			"/hlist                     - show available hidden channels",
			"/join <#channel>           - join/create a channel",
			"/hjoin <#channel>          - join/create hidden channel",
			"/history <#channel>        - return previous messages",
			"/license					- view license agreement",
			"/logoff                    - logoff"})

}

func do_license(clt *Client, args string) {

	clt.SayN(">/license>",
		[]string{"cherrysrv Copyright (C) 2023 Roger Sen roger.sen@gmail.com",
			"This program comes with ABSOLUTELY NO WARRANTY",
			"This is free software, and you are welcome to redistribute it",
			"under certain conditions (gpl3); review https://www.gnu.org/licenses/gpl-3.0.txt for details."})

}

// show internal timer
func do_clock(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/clock>0>/clock requires you to be logged")

		return
	}

	clt.Say(">/clock>0>%d", TIME)
}

// show software version
func do_version(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/version>0>/version requires you to be logged")

		return
	}

	clt.Say(">/version>0>%s", STRINGVER)
}

// show server uptime
func do_uptime(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/uptime>0>/uptime requires you to be logged")

		return
	}

	clt.Say(">/uptime>0>%s", uptime(STARTEDON))
}

// count number of users logged
func do_nusers(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/nusers>0>/nusers requires you to be logged")

		return
	}

	/* Do command */

	if no(args) {
		total_nusers(clt)
		return
	}

	channelName, _ := split2(args, " ")

	channel_nusers(clt, channelName)

}

func total_nusers(clt *Client) {
	NumUsers := 0

	CountUsers := func(key string, v *Client) bool {

		// we don't want to count users that are logging out
		if v.Status.Load() != USER_LOGGINOUT {
			NumUsers++
		}

		return true
	}

	CLIENTS.Range(CountUsers)

	clt.Say(">/nusers>0>%d", NumUsers)
}

func channel_nusers(clt *Client, channelName string) {

	channel, ok := CHANNELS.Load(channelName)

	if !ok {
		clt.Say(">/users %s>0>%s is not a valid channel", channelName, channelName)
		return
	}

	clt.Say(">/users %s>0>%d", channel, channel.Count())
}

// talk to other logged users
func do_say(clt *Client, args string) {

	channelName, message := split2(args, " ")

	channel, ok := CHANNELS.Load(channelName)

	if !ok {
		clt.Say("%s is not a valid channel", channelName)
		return
	}

	channel.Say(clt, "%s", message)

}

// get previous n messages written to channel
func do_history(clt *Client, channelName string) {

	channel, ok := CHANNELS.Load(channelName)

	if !ok {
		clt.Say("%s is not a valid channel", channelName)
		return
	}

	var history []string
	for _, p := range channel.history.readAll() {
		if p != nil {
			history = append(history, strings.TrimPrefix(*p, ">"))
		}
	}

	clt.SayN(">/history>", history)
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

	if no(args) {
		total_users(clt)
		return
	}

	channelName, _ := split2(args, " ")

	channel_users(clt, channelName)

}

// show total number of users in the system
func total_users(clt *Client) {
	var out []string

	print_key := func(key string, c *Client) bool {

		if c.Status.Load() != USER_LOGGINOUT {
			out = append(out, key)
		}
		return true
	}

	CLIENTS.Range(print_key)

	clt.SayN(">/users>", out)
}

// show total number of users in channelName
func channel_users(clt *Client, channelName string) {

	channel, ok := CHANNELS.Load(channelName)

	if !ok {
		clt.Say(">/users %s>0>%s is not a valid channel", channelName, channelName)
		return
	}

	clt.SayN(">/users "+channel.Name+">", channel.ClientNames())

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
		clt.Say(">/login>0>%s is not a valid username because %s", args, err.Error())
		WARN.Printf("user %s unable to login due to: %s", args, err.Error())

		return
	}

	_, ok := CLIENTS.Load(username)

	if ok {
		clt.Say(">/login>0>%s is already taken, please select another @name", username)
		return
	}

	/* Do command */

	oldName := clt.Name

	clt.Name = username
	clt.Status.Store(USER_LOGGED)
	CLIENTS.Store(clt.Name, clt)
	CLIENTS.Delete(oldName)

	mainChannel, _ := CHANNELS.Load("#main")

	mainChannel.addClient(clt)
	clt.Channels = append(clt.Channels, mainChannel)

	/* Update player */

	clt.Say(">/login>0>you're now %s", clt)
	clt.UpdateInMain(">!login>%s has joined the server", clt)

	INFO.Printf("%s has logged in as %s", oldName, clt)
}

// logoff user
func do_logoff(clt *Client, args string) {

	clt.Status.Store(USER_LOGGINOUT)

	clt.Say(">/logoff>0>Goodbye %s", clt)

	clt.UpdateInMain(">!logoff>%s is leaving", clt)

	INFO.Printf("%s logged off (%s)", clt, clt.conn.RemoteAddr())

	clt.Close()

	runtime.Goexit()
}

// show name of logged user
func do_who(clt *Client, args string) {
	clt.Say(">/who>0>%s", clt)
}

func do_join(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/join>0>/join requires you to be logged")

		return
	}

	if no(args) {
		clt.Say(">/join>0>/join <#channel>")

		return
	}

	channelName, _ := split2(args, " ")

	// Multiple users can add at the same time, or a channel can be deleted from Map
	// after the initial lookup.
	CHNMTX.Lock()
	defer CHNMTX.Unlock()

	channel, ok := CHANNELS.Load(channelName)

	if ok {
		channel.addClient(clt)
		clt.Channels = append(clt.Channels, channel)
		channel.Say(clt, "joined the channel")

		return
	}

	channelName, err := ValidChannelname(args)

	if err != nil {
		clt.Say(">/join>0>%s is not a valid channelname because %s", args, err.Error())
		WARN.Printf("user %s unable to create channel %s  due to: %s", clt, args, err.Error())

		return
	}

	NewChannel := newChannel(channelName, false)
	NewChannel.addClient(clt)
	clt.Channels = append(clt.Channels, NewChannel)

	CHANNELS.Store(NewChannel.Key(), NewChannel)
	DEBUG.Printf("adding %s to CHANNELS", NewChannel)

	clt.Say(">/join>0>%s joined %s", clt, NewChannel)
}

func do_hjoin(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/hjoin>0>/hjoin requires you to be logged")

		return
	}

	if no(args) {
		clt.Say(">/hjoin>0>/hjoin <#channel>")

		return
	}

	channelName, _ := split2(args, " ")

	// Multiple users can add at the same time, or a channel can be deleted from Map
	// after the initial lookup.
	CHNMTX.Lock()
	defer CHNMTX.Unlock()

	channel, ok := CHANNELS.Load(channelName)

	if ok {
		channel.addClient(clt)
		clt.Channels = append(clt.Channels, channel)
		channel.Say(clt, "hjoined the channel")

		return
	}

	channelName, err := ValidChannelname(args)

	if err != nil {
		clt.Say(">/hjoin>0>%s is not a valid channelname because %s", args, err.Error())
		WARN.Printf("user %s unable to create hchannel %s  due to: %s", clt, args, err.Error())

		return
	}

	NewChannel := newChannel(channelName, true)
	NewChannel.addClient(clt)
	clt.Channels = append(clt.Channels, NewChannel)

	CHANNELS.Store(NewChannel.Key(), NewChannel)
	DEBUG.Printf("adding %s to CHANNELS", NewChannel)

	clt.Say(">/hjoin>0>%s hjoined %s", clt, NewChannel)

}

func do_leave(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/leave>0>/join requires you to be logged")

		return
	}

	if no(args) {
		clt.Say(">/leave>0>/leave <#channel>")

		return
	}

	channelName, _ := split2(args, " ")

	// Multiple users can add at the same time, or a channel can be deleted from Map
	// after the initial lookup.
	CHNMTX.Lock()
	defer CHNMTX.Unlock()

	channel, ok := CHANNELS.Load(channelName)

	if ok {

		if !channel.findClient(clt) {
			clt.Say(">/leave>0>you're not in channel %s", channelName)
			return
		}

		channel.Say(clt, "left the channel")
		channel.removeClient(clt)
		clt.RemoveChannel(channel)

		return
	}

	clt.Say(">/leave>0>channel %s does not exist", channelName)

}

// show non hidden channels
func do_list(clt *Client, args string) {

	if !clt.isLogged() {
		clt.Say(">/list>0>/list requires you to be logged")

		return
	}

	/* Do command */

	var out []string

	print_key := func(key string, channel *Channel) bool {

		if !channel.isHidden() {
			out = append(out, key)
		}
		return true
	}

	CHANNELS.Range(print_key)

	sort.Strings(out)

	clt.SayN(">/list>", out)
}
