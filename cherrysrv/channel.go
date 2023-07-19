package main

import (
	"fmt"
	"sort"
	"sync"
)

type Channel struct {
	clients      []*Client // clients in the channel.
	Name         string    // Name of the channel (incl #)
	hidden       bool
	closeOnEmpty bool // only #main should have this as false
	sync.RWMutex      // for adding/removing client connections
}

func newChannel(name string, hiddenChannel bool) *Channel {
	return &Channel{
		clients:      []*Client{},
		Name:         name,
		hidden:       hiddenChannel,
		closeOnEmpty: true,
		RWMutex:      sync.RWMutex{},
	}
}

func NewChannelMain(name string) *Channel {
	return &Channel{
		clients:      []*Client{},
		Name:         name,
		hidden:       false,
		closeOnEmpty: false,
		RWMutex:      sync.RWMutex{},
	}
}

// return the key to index the channel
func (c *Channel) Key() string {
	return c.Name
}

// return the name of the channel
func (c *Channel) String() string {
	return c.Name
}

func (c *Channel) isHidden() bool {
	return c.hidden
}

func (c *Channel) Count() int {
	return len(c.clients)
}

func (c *Channel) ClientNames() (output []string) {
	c.RLock()
	defer c.RUnlock()

	for _, client := range c.clients {
		if client.Status.Load() != USER_LOGGINOUT {
			clientname := client.Name
			output = append(output, clientname)
		}
	}

	sort.Strings(output)

	return output
}

func (c *Channel) findClient(client *Client) bool {
	c.RLock()
	defer c.RUnlock()

	for i := 0; i < len(c.clients); i++ {
		if c.clients[i].Name == client.Name {
			return true
		}
	}

	return false
}

// TODO: Review subtle bug:
// if addClient is blocked because removeClient is working
// and removeClient leaves channel with 0 elements removing it
// from CHANNELS directory completely, addClient will add a client to a
// removed channel.
func (channel *Channel) addClient(newClient *Client) {
	channel.Lock()
	defer channel.Unlock()

	channel.clients = append(channel.clients, newClient)
}

func (channel *Channel) removeClient(client *Client) bool {
	channel.Lock()
	defer channel.Unlock()

	len := len(channel.clients)

	// len(c.clients) = 0 or 1, we manage them as special cases
	// let's treat them manually:

	if len == 0 {
		DEBUG.Printf("%s has 0 clients and this should not be possible", channel)
		return false
	}

	if len == 1 && channel.clients[0].Name == client.Name {
		channel.clients = []*Client{}

		if channel.closeOnEmpty {
			DEBUG.Printf("%s has now 0 clients, removing it from the directory", channel)

			CHANNELS.Delete(channel.Name)
		}

		return true
	}

	if len == 1 && channel.clients[0].Name != client.Name {
		return false
	}

	// len(c.clients) >= 2
	// we loop through all the slice, NOT starting in pos=2

	for i := 0; i < len; i++ {
		if channel.clients[i] == client {
			channel.clients[i] = channel.clients[len-1] // TODO: confirm is copying pointer, not content
			channel.clients = channel.clients[:len-1]

			return true
		}
	}

	return false
}

func (channel *Channel) Say(from *Client, format string, args ...interface{}) {

	message := fmt.Sprintf(format, args...)

	if len(message) == 0 {
		return
	}

	channel.write(from, ">"+channel.Name+">"+from.Name+">"+message+"\n")
}

func (c *Channel) write(from *Client, message string) {
	c.RLock()
	defer c.RUnlock()

	len := len(c.clients)

	if len == 0 {
		return
	}

	for i := 0; i < len; i++ {
		if c.clients[i].isLogged() { // TODO: we should be able to remove this check. If is in a channel IT MUST be logged.
			c.clients[i].write(message)
		}
	}

}
