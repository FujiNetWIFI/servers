package main

import (
	"fmt"
	"strings"
)

// /who @john -> ["who", "@john"]
// anything else -> ["say", line]

func parse(line string) (command string, args string) {

	if len(line) == 0 {
		return command, args
	}

	if line[0] == '/' {
		split := strings.SplitN(line[1:], " ", 2) // we remove the '/' at pos 0

		if len(split) == 1 {
			return strings.Trim(split[0], " \r\n"), args
		}

		return strings.Trim(split[0], " \r\n"), strings.Trim(split[1], " \r\n")
	}

	return "say", strings.Trim(line, " \r\n")

}

func exec(clt *Client, command string, args string) (string, error) {

	_, ok := COMMANDS[command]

	if ok {
		COMMANDS[command](clt, args)

		return command, nil
	}

	return command, fmt.Errorf("command %s not found", command)
}
