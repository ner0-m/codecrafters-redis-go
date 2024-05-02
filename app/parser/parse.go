package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	Error  = '-'
	Status = '+'
	Int    = ':'
	Bulk   = '$'
	Array  = '*'
)

type Type byte

type Message struct {
	Raw  string
	Data []string
}

func PeekMsgType(tokens []string) Type {
	return Type(tokens[0][0])
}

func ParseArray(msgType Type, cur string, reader *bufio.Reader) (string, []string) {
	n, err := strconv.Atoi(cur[1 : len(cur)-2])

	if err != nil {
		fmt.Printf("Error parsing array length: %s\n", err.Error())
		return "", nil
	}

	var msg []string
	raw := cur

	i := 0
	for i < n {
		data, err := reader.ReadString('\n')

		if err != nil {
			fmt.Printf("Error reading for array: %s\n", err.Error())
			continue
		}

		r, m := ParseMsg(data, reader)
		raw = raw + r
		msg = append(msg, m...)

		i++
	}

	return raw, msg
}

func ParseMsg(cur string, reader *bufio.Reader) (string, []string) {
	msgType := Type(cur[0])
	switch msgType {
	case Error, Status, Int:
		return cur, []string{cur[1 : len(cur)-2]}
	case Bulk:
		n, err := strconv.Atoi(cur[1 : len(cur)-2])

		if err != nil {
			fmt.Printf("Error reading for length of bulk string: %s\n", err.Error())
			return "", nil
		}

		// Read n bytes from connection
		buf := make([]byte, n)
		n, err = io.ReadFull(reader, buf)

		if err != nil {
			fmt.Printf("Could not read bulk string: %s\n", err.Error())
			return "", nil
		}

		// If the next two bytes are \r\n, read it and discard it
		var tmp []byte
		tmp, err = reader.Peek(2)
		if err == nil && tmp[0] == '\r' && tmp[1] == '\n' {
			reader.Read(tmp)

			buf = append(buf, '\r', '\n')
			n += 2
		}

		raw := strings.Join([]string{cur, string(buf[:n])}, "")
		return raw, []string{string(buf[:n-2])}
	case Array:
		return ParseArray(msgType, cur, reader)
	}
	return "", nil
}
