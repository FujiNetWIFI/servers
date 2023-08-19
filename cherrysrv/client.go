package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
)

// client status
const (
	USER_NOTLOGGED = 1 // Player connected and mud waiting for login.
	USER_LOGGED    = 2 // Player autheticated and currently chatting.
	USER_LOGGINOUT = 4 // Player being cleaned up, it won't accept any string sent to them.
)

// Client connection storing basic client data
type Client struct {
	conn   net.Conn // network connection interface.
	Name   string   // Name of the user.
	Status atomic.Int32
}

func (c *Client) String() string {
	return c.Name
}

func newClient(conn net.Conn) *Client {

	client := &Client{
		conn: conn,
		Name: gensym("@Anon"),
	}
	client.Status.Store(USER_NOTLOGGED)

	INFO.Printf("%s has connected (%s)", client.Name, client.conn.RemoteAddr())

	CLIENTS.Store(client.Key(), client)

	return client
}

func (c *Client) Key() string {
	return c.Name
}

// Close a client connection following ws protocol plus removing the internal handlers in the mud.
func (clt *Client) Close() {

	clt.RemoveMeFromAllChannels()
	clt.conn.Close()
	CLIENTS.Delete(clt.Name)
}

// main client loop that process client's messages
// https://github.com/uber-go/ratelimit
func (clt *Client) clientLoop() {

	clt.Say(">#main>!welcome>welcome to cherry server %s # %s", clt.Name, STRINGVER)

	for {

		// we don't want to read from a socket that is logging out
		if clt.Status.Load() == USER_LOGGINOUT {
			return
		}

		line, err := clt.read()
		if err != nil {
			INFO.Printf("%s disconnected (%s)", clt, clt.conn.RemoteAddr())
			clt.UpdateInMain(">!disconnect>%s disconnected", clt)
			clt.Close()

			return
		}

		command, args := parse(line)

		if no(command) { // line was empty
			continue
		}

		command, err = exec(clt, command, args)

		if err != nil {
			clt.Say(">/%s>0>command %s does not exist", command, command)

			continue // no really needed, but for consistency.
		}
	}
}

// Send a message to the client
func (clt *Client) Say(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	clt.write(line + "\n")
}

// Send len(Lines) with a lead message to the client
func (clt *Client) SayN(lead string, Lines []string) {

	NumElems := len(Lines)

	if NumElems == 0 {
		return
	}

	var output strings.Builder
	NumElems -= 1 // we count from NumElems-1 to 0

	for _, line := range Lines {
		text := fmt.Sprintf("%s%d>%s\n", lead, NumElems, line)

		text = shorten255(text)

		output.WriteString(text)
		NumElems -= 1
	}

	clt.writeNoLimit(output.String())

}

// write a message to the client. Limited to 255 chars.
func (clt *Client) write(line string) (n int, err error) {

	if len(line) == 0 {
		return
	}

	line = shorten255(line)

	return clt.writeNoLimit(line)
}

// writeNoLimit a message to the client. Unlimited length.
func (clt *Client) writeNoLimit(line string) (n int, err error) {

	if len(line) == 0 {
		return
	}

	DataLength, err := clt.conn.Write([]byte(line))

	if err != nil {
		DEBUG.Printf("%s.write() failed with err: %s", clt, err)
	}

	return DataLength, err
}

// check if client is logged
func (clt *Client) isLogged() bool {
	return clt.Status.Load() == USER_LOGGED
}

// Read message sent by client, limited to 255 chars
func (client *Client) read() (string, error) {

	netData, err := bufio.NewReader(client.conn).ReadString('\n')

	if err != nil {
		DEBUG.Printf("%s.read() failed with err: %s", client, err)
	}

	netData = shorten255(netData)

	return netData, nil
}

// to be used by the server, send a message to everyone connected (including the sender)
// when there's no client associated (CRTL-C)

func Broadcast(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	broadcast := func(key string, clt *Client) bool {
		clt.Say(line + "\n")
		return true
	}

	CLIENTS.Range(broadcast)
}

// to be user by the server, send a message to everyone connected to the main channel (excluding the sender)
func (clt *Client) UpdateInMain(format string, args ...interface{}) {

	line := fmt.Sprintf(format, args...)

	broadcast := func(key string, client *Client) bool {

		if clt == client { // we don't want to send the message to us
			return true
		}

		client.write(">#main" + line + "\n")
		return true
	}

	CLIENTS.Range(broadcast)
}

// delete me from all the channels. This can be optimised in the future.
func (clt *Client) RemoveMeFromAllChannels() {

	removeClient := func(key string, channel *Channel) bool {

		channel.removeClient(clt)

		return true
	}

	CHANNELS.Range(removeClient)
}
