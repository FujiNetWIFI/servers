package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
)

type player_status int

const (
	USER_NOTLOGGED player_status = iota // Player connected and mud waiting for login.
	USER_LOGGED                         // Player autheticated and currently playing.
	USER_LOGGINOUT                      // Player being cleaned up, it won't accept any string sent to them.
)

// Client connection storing basic PC data
type Client struct {
	conn       *net.TCPConn  // tcpsocket connection.
	name       string        // Name of the user.
	status     player_status // Current status of player's connection (Not logged, Playing and Logging out)
	conn_mutex sync.Mutex    // gorilla websocket does not allow concurrent writes, use a mutex for writing conn
}

func newClient(conn *net.TCPConn) *Client {

	client := &Client{
		conn:   conn,
		name:   gensym("Anon"),
		status: USER_NOTLOGGED,
	}

	INFO.Printf("%s has connected (%s)", client.name, client.conn.RemoteAddr())

	CLIENTS.Store(client.name, client)

	return client
}

// Close a client connection following ws protocol plus removing the internal handlers in the mud.
func (clt *Client) Close() {

	clt.status = USER_LOGGINOUT
	clt.conn.Close()
	CLIENTS.Delete(clt.name)
}

// main client loop that process client's messages
// https://github.com/uber-go/ratelimit
func (clt *Client) clientLoop() {

	clt.OKPrintf("welcome to cherry server %s # %s", clt.name, STRINGVER)

	for {

		// we don't want to read from a socket that is logging out
		if clt.status == USER_LOGGINOUT {
			return
		}

		line, err := clt.read()
		if err != nil {
			INFO.Printf("%s disconnected (%s)", clt.name, clt.conn.RemoteAddr())
			clt.Close()

			return
		}

		command, args := parse(line)

		if no(command) { // line was empty
			continue
		}

		command, err = exec(clt, command, args)

		if err != nil {
			clt.FAILPrintf("command %s does not exist", command)

			continue // no really needed, but for consistency.
		}
	}
}

// func (clt *Client) OKPrintf(Line string) {
func (clt *Client) OKPrintf(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	clt.Write(">info>srv>" + line + "\n")
}

func (clt *Client) FAILPrintf(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	clt.Write(">info>srv>" + line + "\n")
}

func (clt *Client) OKPrintfN(Lines []string) {

	for _, line := range Lines {
		clt.Write(">info>srv>" + line + "\n")
	}

}

// Write a message to the client to be sent back to the player via websocket
func (clt *Client) Write(line string) (n int, err error) {

	data := []byte(line)

	length := len(data)

	if length != 0 {
		clt.conn_mutex.Lock() // TODO: do we need this lock? We needed if for websocket.
		clt.conn.Write(data)
		clt.conn_mutex.Unlock()
	}

	return length, nil

}

// check if client is logged

func (clt *Client) isLogged() bool {
	return clt.status == USER_LOGGED
}

func (client *Client) read() (string, error) {

	netData, err := bufio.NewReader(client.conn).ReadString('\n')

	if len(netData) > 255 {
		netData = netData[:255]
	}

	return netData, err
}

// to be used by the server, send a message to everyone connected (including the sender)
func (clt *Client) Broadcast(format string, args ...interface{}) {

	Broadcast(format, args...)
}

// to be used by the server, send a message to everyone connected (including the sender)
// when there's no client associated (CRTL-C)

func Broadcast(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	OobBroadcast := func(key string, clt *Client) bool {
		clt.OKPrintf(line + "\n")
		return true
	}

	CLIENTS.Range(OobBroadcast)
}

// to be user by the server, send a message to everyone connected (excluding the sender)
func (clt *Client) BroadcastButMe(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	OobBroadcast := func(key string, client *Client) bool {

		if clt == client { // we don't want to send the message to us
			return true
		}

		client.Write(line + "\n")
		return true
	}

	CLIENTS.Range(OobBroadcast)
}

// to be user by the users, send a message to everyone USER_LOGGED but the client.
func (clt *Client) SayToAllButMe(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	OobBroadcast := func(key string, client *Client) bool {

		if clt == client { // we don't want to send the message to us
			return true
		}

		if clt.isLogged() { // we want to send the message only to
			client.Write(">main>" + clt.name + ">" + line + "\n")
		}

		return true
	}

	CLIENTS.Range(OobBroadcast)
}
