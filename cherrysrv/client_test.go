package main

import (
	"bufio"
	"fmt"
	"net"
	"testing"
)

// TestClient is a set of ordered happy path tests
func TestClient(t *testing.T) {
	init_logger()
	init_commands()
	main_channel := NewChannelMain("#main")
	CHANNELS.Store(main_channel.Key(), main_channel)

	server, client := net.Pipe()
	connWrapper := bufio.NewReadWriter(bufio.NewReader(client), bufio.NewWriter(client))

	nc := newClient(server)
	go nc.clientLoop()

	if res, _, err := connWrapper.ReadLine(); err == nil {
		fmt.Println(string(res))
	}

	username := "@tester"
	chan1 := "#test"
	chan2 := "#test2"

	clientTests := []struct {
		input    []byte
		expected []string
	}{
		{[]byte("/who\n"), []string{fmt.Sprintf(">/who>0>%s", nc.Name)}},
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
		client.Write(test.input)

		for i, ex := range test.expected {
			if res, _, err := connWrapper.ReadLine(); err != nil {
				t.Errorf("failed read, expected %s", test.expected[i])
			} else if string(res) != ex {
				t.Errorf("got %s, expected %s", string(res), ex)
			}
		}

	}
}
