package main

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"
)

func genClient() (c *Client, out net.Conn, in *bufio.Reader) {
	server, out := net.Pipe()

	in = bufio.NewReader(out)

	c = newClient(server)
	go c.clientLoop()

	if res, _, err := in.ReadLine(); err == nil {
		fmt.Println(string(res))
	}

	return
}

// TestClient is a set of ordered happy path tests
func TestSingleClient(t *testing.T) {
	// configure test server
	init_logger()
	init_commands()
	main_channel := NewChannelMain("#main")
	CHANNELS.Store(main_channel.Key(), main_channel)

	// generate clients
	c1, out, in := genClient()

	username := "@tester"
	chan1 := "#test"
	chan2 := "#test2"

	clientTests := []struct {
		name     string
		input    []byte
		expected []string
	}{
		{"Anon Whois Test", []byte("/who\n"), []string{fmt.Sprintf(">/who>0>%s", c1.Name)}},
		{"Fail Channel Join Test", []byte("/join #test\n"), []string{">/join>0>/join requires you to be logged"}},
		{"Fail User Count Test", []byte("/nusers\n"), []string{">/nusers>0>/nusers requires you to be logged"}},
		{"Fail User List Test", []byte("/users\n"), []string{">/users>0>/users requires you to be logged"}},
		{"Login Help Test", []byte("/login\n"), []string{">/login>0>/login <account>"}},
		{"Login Test", []byte(fmt.Sprintf("/login %s\n", username)), []string{fmt.Sprintf(">/login>0>you're now %s", username)}},
		{"Duplicate Login Test", []byte("/login @tester2\n"), []string{">/login>0>you're already logged in"}},
		{"User Count Test", []byte("/nusers\n"), []string{">/nusers>0>1"}},
		{"User List Test", []byte("/users\n"), []string{">/users>0>@tester"}},
		{"Channel Join Help Test", []byte("/join\n"), []string{">/join>0>/join <#channel>"}},
		{"Channel Join Test", []byte(fmt.Sprintf("/join %s\n", chan1)), []string{fmt.Sprintf(">/join>0>%s joined %s", username, chan1)}},
		{"Channel Say Test", []byte(fmt.Sprintf("/say %s hello\n", chan1)), []string{fmt.Sprintf(">%s>%s>hello", chan1, username)}},
		{"Channel Say Test #2", []byte(fmt.Sprintf("/say %s goodbye\n", chan1)), []string{fmt.Sprintf(">%s>%s>goodbye", chan1, username)}},
		{"Invalid Channel User Count Test", []byte("/nusers #bigapple\n"), []string{">/users #bigapple>0>#bigapple is not a valid channel"}},
		{"Channel User Count Test", []byte(fmt.Sprintf("/nusers %s\n", chan1)), []string{fmt.Sprintf(">/users %s>0>1", chan1)}},
		{"Channel User Test", []byte(fmt.Sprintf("/users %s\n", chan1)), []string{fmt.Sprintf(">/users %s>0>%s", chan1, username)}},
		{"Channel List Test", []byte("/list\n"), []string{fmt.Sprintf(">/list>1>%s", main_channel.Name), fmt.Sprintf(">/list>0>%s", chan1)}},
		{"Channel Leave Test", []byte(fmt.Sprintf("/leave %s\n", chan1)), []string{fmt.Sprintf(">%s>%s>left the channel", chan1, username)}},
		{"Channel List Test #2 (empty channel cleanup)", []byte("/list\n"), []string{fmt.Sprintf(">/list>0>%s", main_channel.Name)}},
		{"Hidden Channel Join Test", []byte(fmt.Sprintf("/hjoin %s\n", chan2)), []string{fmt.Sprintf(">/hjoin>0>%s hjoined %s", username, chan2)}},
		{"Channel List Test #3 (ignore hidden)", []byte("/list\n"), []string{fmt.Sprintf(">/list>0>%s", main_channel.Name)}},
		{"Hidden Channel Leave Test", []byte(fmt.Sprintf("/leave %s\n", chan2)), []string{fmt.Sprintf(">%s>%s>left the channel", chan2, username)}},
		{"Logoff Test", []byte("/logoff\n"), []string{fmt.Sprintf(">/logoff>0>Goodbye %s", username)}},
	}

	for _, test := range clientTests {
		// send data to the server
		out.Write(test.input)

		// retrieve all data returned by server
		rets := make(chan []string)
		go fullRead(in, out, rets)
		res := <-rets

		if len(res) != len(test.expected) {
			t.Errorf("%s got %v, expected %v", test.name, res, test.expected)
		} else {
			// compare each line returned and validate that it matches expected
			for i, ex := range test.expected {
				if string(res[i]) != ex {
					t.Errorf("%s got %s, expected %s", test.name, string(res[i]), ex)
				}
			}
		}
	}
}

// fullRead is a function that takes a reader, a conn and a channel and reads data
// until a timeout deadline is met. The results are fed to the input channel. Reading this way
// prevents a deadlock of the net.Pipe
func fullRead(buff *bufio.Reader, conn net.Conn, c chan []string) {
	conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	var s []string
	for {
		if res, _, err := buff.ReadLine(); err != nil {
			break
		} else {
			s = append(s, string(res))
		}
	}
	c <- s
}
