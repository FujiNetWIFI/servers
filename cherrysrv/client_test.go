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

type multiCase struct {
	i *bufio.Reader
	c net.Conn
	s []string
}

// TestClient is a set of ordered happy path tests
func TestClient(t *testing.T) {
	init_logger()
	init_commands()
	main_channel := NewChannelMain("#main")
	CHANNELS.Store(main_channel.Key(), main_channel)

	c1, out, in := genClient()

	username := "@tester"
	chan1 := "#test"
	chan2 := "#test2"

	clientTests := []struct {
		input    []byte
		expected []string
	}{
		{[]byte("/who\n"), []string{fmt.Sprintf(">/who>0>%s", c1.Name)}},
		{[]byte("/join #test\n"), []string{">/join>0>/join requires you to be logged"}},
		{[]byte("/nusers\n"), []string{">/nusers>0>/nusers requires you to be logged"}},
		{[]byte("/users\n"), []string{">/users>0>/users requires you to be logged"}},
		{[]byte("/login\n"), []string{">/login>0>/login <account>"}},
		{[]byte(fmt.Sprintf("/login %s\n", username)), []string{fmt.Sprintf(">/login>0>you're now %s", username)}},
		{[]byte("/login @tester2\n"), []string{">/login>0>you're already logged in"}},
		{[]byte("/nusers\n"), []string{">/nusers>0>1"}},
		{[]byte("/users\n"), []string{">/users>0>@tester"}},
		{[]byte("/join\n"), []string{">/join>0>/join <#channel>"}},
		{[]byte(fmt.Sprintf("/join %s\n", chan1)), []string{fmt.Sprintf(">/join>0>%s joined %s", username, chan1)}},
		{[]byte(fmt.Sprintf("/say %s hello\n", chan1)), []string{fmt.Sprintf(">%s>%s>hello", chan1, username)}},
		{[]byte(fmt.Sprintf("/say %s goodbye\n", chan1)), []string{fmt.Sprintf(">%s>%s>goodbye", chan1, username)}},
		{[]byte("/nusers #bigapple\n"), []string{">/users #bigapple>0>#bigapple is not a valid channel"}},
		{[]byte(fmt.Sprintf("/nusers %s\n", chan1)), []string{fmt.Sprintf(">/users %s>0>1", chan1)}},
		{[]byte(fmt.Sprintf("/users %s\n", chan1)), []string{fmt.Sprintf(">/users %s>0>%s", chan1, username)}},
		{[]byte(fmt.Sprintf("/history %s\n", chan1)), []string{fmt.Sprintf(">/history>1>%s>%s>hello", chan1, username), fmt.Sprintf(">/history>0>%s>%s>goodbye", chan1, username)}},
		{[]byte("/list\n"), []string{fmt.Sprintf(">/list>1>%s", main_channel.Name), fmt.Sprintf(">/list>0>%s", chan1)}},
		{[]byte(fmt.Sprintf("/leave %s\n", chan1)), []string{fmt.Sprintf(">%s>%s>left the channel", chan1, username)}},
		{[]byte("/list\n"), []string{fmt.Sprintf(">/list>0>%s", main_channel.Name)}},
		{[]byte(fmt.Sprintf("/hjoin %s\n", chan2)), []string{fmt.Sprintf(">/hjoin>0>%s hjoined %s", username, chan2)}},
		{[]byte("/list\n"), []string{fmt.Sprintf(">/list>0>%s", main_channel.Name)}},
		{[]byte(fmt.Sprintf("/leave %s\n", chan2)), []string{fmt.Sprintf(">%s>%s>left the channel", chan2, username)}},
		{[]byte("/logoff\n"), []string{fmt.Sprintf(">/logoff>0>Goodbye %s", username)}},
	}

	for _, test := range clientTests {
		out.Write(test.input)

		for i, ex := range test.expected {
			if res, _, err := in.ReadLine(); err != nil {
				t.Errorf("failed read, expected %s", test.expected[i])
			} else if string(res) != ex {
				t.Errorf("got %s, expected %s", string(res), ex)
			}
		}

	}
}

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

// TestMultipleClients
func TestMultipleClients(t *testing.T) {
	init_logger()
	init_commands()
	main_channel := NewChannelMain("#main")
	CHANNELS.Store(main_channel.Key(), main_channel)

	_, out1, in1 := genClient()
	_, out2, in2 := genClient()
	_, out3, in3 := genClient()

	username1 := "@tester1"
	username2 := "@tester2"
	username3 := "@tester3"

	chan1 := "#test"

	clientTests := []struct {
		o        net.Conn
		input    []byte
		expected []multiCase
	}{
		{out1, []byte(fmt.Sprintf("/login %s\n", username1)),
			[]multiCase{{in1, out1, []string{fmt.Sprintf(">/login>0>you're now %s", username1)}}}},

		{out2, []byte(fmt.Sprintf("/login %s\n", username1)),
			[]multiCase{{in2, out2, []string{fmt.Sprintf(">/login>0>%s is already taken, please select another @name", username1)}}}},

		{out2, []byte(fmt.Sprintf("/login %s\n", username2)),
			[]multiCase{{in2, out2, []string{fmt.Sprintf(">/login>0>you're now %s", username2)}},
				{in1, out1, []string{fmt.Sprintf(">#main>!login>%s has joined the server", username2)}}}},

		{out3, []byte(fmt.Sprintf("/login %s\n", username3)),
			[]multiCase{{in3, out3, []string{fmt.Sprintf(">/login>0>you're now %s", username3)}},
				{in2, out2, []string{fmt.Sprintf(">#main>!login>%s has joined the server", username3)}},
				{in1, out1, []string{fmt.Sprintf(">#main>!login>%s has joined the server", username3)}}}},

		{out1, []byte(fmt.Sprintf("/join %s\n", chan1)),
			[]multiCase{{in1, out1, []string{fmt.Sprintf(">/join>0>%s joined %s", username1, chan1)}}}},

		{out2, []byte(fmt.Sprintf("/join %s\n", chan1)),
			[]multiCase{{in2, out2, []string{fmt.Sprintf(">%s>%s>joined the channel", chan1, username2)}},
				{in1, out1, []string{fmt.Sprintf(">%s>%s>joined the channel", chan1, username2)}}}},

		{out3, []byte(fmt.Sprintf("/join %s\n", chan1)),
			[]multiCase{{in3, out3, []string{fmt.Sprintf(">%s>%s>joined the channel", chan1, username3)}},
				{in2, out2, []string{fmt.Sprintf(">%s>%s>joined the channel", chan1, username3)}},
				{in1, out1, []string{fmt.Sprintf(">%s>%s>joined the channel", chan1, username3)}}}},

		{out1, []byte(fmt.Sprintf("/say %s hello\n", chan1)),
			[]multiCase{{in3, out3, []string{fmt.Sprintf(">%s>%s>hello", chan1, username1)}},
				{in2, out2, []string{fmt.Sprintf(">%s>%s>hello", chan1, username1)}},
				{in1, out1, []string{fmt.Sprintf(">%s>%s>hello", chan1, username1)}}}},

		{out1, []byte("/logoff\n"),
			[]multiCase{{in1, out1, []string{fmt.Sprintf(">/logoff>0>Goodbye %s", username1)}},
				{in2, out2, []string{fmt.Sprintf(">#main>!logoff>%s is leaving", username1)}},
				{in3, out3, []string{fmt.Sprintf(">#main>!logoff>%s is leaving", username1)}}}},
		{out2, []byte("/logoff\n"),
			[]multiCase{{in2, out2, []string{fmt.Sprintf(">/logoff>0>Goodbye %s", username2)}},
				{in3, out3, []string{fmt.Sprintf(">#main>!logoff>%s is leaving", username2)}}}},
		{out3, []byte("/logoff\n"),
			[]multiCase{{in3, out3, []string{fmt.Sprintf(">/logoff>0>Goodbye %s", username3)}}}},
	}

	for _, test := range clientTests {
		test.o.Write(test.input)

		var rets []chan []string
		for i, ex := range test.expected {
			rets = append(rets, make(chan []string))
			go fullRead(ex.i, ex.c, rets[i])
		}
		for i, ex := range test.expected {
			res := <-rets[i]
			if len(ex.s) != len(res) {
				t.Errorf("got %v, expected %v", res, ex.s)
			} else {
				for i, s := range ex.s {
					if s != res[i] {
						t.Errorf("got %s, expected %s", res[i], s)
					}
				}
			}
		}

	}
}
