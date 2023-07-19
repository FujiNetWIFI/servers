package main

import (
	"fmt"
)

// /who @john -> ["who", "@john"]
// #channel word1 word2 word3 -> ["say", "#channel", line]

func parse(line string) (command string, args string) {

	line = trim(line)

	if len(line) == 0 {
		return command, args
	}

	if line[0] == '/' {
		command, rest := split2(line[1:], " ") // we remove the '/' at pos 0

		return trim(command), trim(rest)

	}

	return "say", trim(line)

}

func exec(clt *Client, command string, args string) (string, error) {

	_, ok := COMMANDS[command]

	if ok {
		COMMANDS[command](clt, args)

		return command, nil
	}

	return command, fmt.Errorf("command %s not found", command)
}
