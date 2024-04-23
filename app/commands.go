package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ping() ([]byte, error) {
	return []byte("+PONG\r\n"), nil
}

func echo(str string) ([]byte, error) {
	return encodeBulk(str), nil
}

func set(cmd []string, store Store) ([]byte, error) {
	if len(cmd) == 3 {
		fmt.Printf("Debug: set %s = %s\n", cmd[1], cmd[2])
		store.Write(cmd[1], cmd[2], nil)
		return []byte("+OK\r\n"), nil
	} else if len(cmd) == 5 && strings.ToLower(cmd[3]) == "px" {
		fmt.Printf("Debug: set %s = %s, %s %s\n", cmd[1], cmd[2], cmd[3], cmd[4])
		d, err := strconv.Atoi(cmd[4])
		if err != nil {
			return nil, errors.New("Could not convert expiery length")
		}
		var dur *time.Duration
		dur = new(time.Duration)
		*dur = time.Duration(d) * time.Millisecond
		store.Write(cmd[1], cmd[2], dur)
		return []byte("+OK\r\n"), nil
	} else {
		return nil, errors.New("Unknown command for set")
	}
}

func get(key string, store Store) ([]byte, error) {
	v, ok := store.Read(key)
	if !ok {
		fmt.Printf("Debug: get %s, not in store\n", key)
		return []byte("$-1\r\n"), nil
	}
	fmt.Printf("Debug: get %s = %s\n", key, v)
	return encodeBulk(v), nil
}

func info(section string) ([]byte, error) {
	if strings.ToLower(section) == "replication" {
		return encodeBulk("# Replication\r\nrole:master\r\n"), nil
	} else {
		return nil, errors.New("Unknown INFO section: '" + section + "'")
	}
}
