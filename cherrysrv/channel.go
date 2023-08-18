package main

import (
	"fmt"
	"sort"
	"sync"
)

// client status
const (
	CHANNEL_WORKING      = 1 // channel created and with users
	CHANNEL_SHUTTINGDOWN = 2 // channel with no users, shutting down
)

type Channel struct {
	clients      []*Client // clients in the channel.
	Name         string    // Name of the channel (incl #)
	hidden       bool
	closeOnEmpty bool // only #main should have this as false
	Status       int  // CHANNEL_WORKING, CHANNEL_SHUTTINGDOWN
	sync.RWMutex      // for adding/removing client connections
}

func newChannel(name string, hiddenChannel bool) *Channel {
	return &Channel{
		clients:      []*Client{},
		Name:         name,
		hidden:       hiddenChannel,
		closeOnEmpty: true,
		Status:       CHANNEL_WORKING,
		RWMutex:      sync.RWMutex{},
	}

}

func NewChannelMain(name string) *Channel {
	return &Channel{
		clients:      []*Client{},
		Name:         name,
		hidden:       false,
		closeOnEmpty: false,
		Status:       CHANNEL_WORKING,

		RWMutex: sync.RWMutex{},
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

// return the client names of the clients currently in this channel
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

// find if a certain client is in this channel
func (c *Channel) contains(client *Client) bool {
	c.RLock()
	defer c.RUnlock()

	for i := 0; i < len(c.clients); i++ {
		if c.clients[i] == client {
			return true
		}
	}

	return false
}

// add client considering if the channel is shutting down
func (channel *Channel) addClient(newClient *Client) bool {
	channel.Lock()
	defer channel.Unlock()

	if channel.Status == CHANNEL_SHUTTINGDOWN {
		return false
	}

	channel.clients = append(channel.clients, newClient)

	return true
}

// remove client and return bool if successful.
// if it's the last channel, remove the channel from the server
func (channel *Channel) removeClient(client *Client) bool {
	channel.Lock()
	defer channel.Unlock()

	len := len(channel.clients)

	// len(c.clients) = 0 or 1, we manage them as special cases

	if len == 0 {
		DEBUG.Printf("%s has 0 clients and this should not be possible", channel)
		return false
	}

	// single client in the group and is the one we want to remove.
	if len == 1 && channel.clients[0] == client {
		if channel.closeOnEmpty {
			channel.Status = CHANNEL_SHUTTINGDOWN
		}
		channel.clients = []*Client{}

		if channel.closeOnEmpty {
			DEBUG.Printf("%s has now 0 clients, removing it from the directory", channel)

			CHANNELS.Delete(channel.Name)
		}

		return true
	}

	// otherwise we loop through all the slice, NOT starting in pos=2,
	// removing it when we find it.
	// Because we have >= 2 clients we do not remove the channel.

	for i := 0; i < len; i++ {
		if channel.clients[i] == client {
			channel.clients[i] = channel.clients[len-1]
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
