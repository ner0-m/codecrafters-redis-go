package main

import (
	"errors"
	"fmt"
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

type Response struct {
	Type  Type
	Raw   []byte
	Data  []byte
	Count int
}

func (r Response) Bytes() []byte {
	return r.Data
}

func (r Response) String() string {
	return string(r.Data)
}

func parseInt(data []byte) error {
	if len(data) == 0 {
		return errors.New("Can not parse empty data as Int")
	}
	_, err := strconv.ParseInt(string(data), 10, 64)

	return err
}

func parseBulk(count int, i int, b []byte) error {
	if count < 0 {
		return errors.New("Can not convert to Bulk string of non-positive length")
	}

	if len(b) < i+count+2 {
		return errors.New("Given bytes and length do not match")
	}

	if b[i+count] != '\r' && b[i+count+1] != '\n' {
		return errors.New("String doesn't end with \\r\\n")
	}

	return nil
}

func encodeBulkNoCrlf(str string) []byte {
	return []byte("$" + strconv.Itoa(len(str)) + "\r\n" + str)
}

func encodeBulk(str string) []byte {
	return []byte("$" + strconv.Itoa(len(str)) + "\r\n" + str + "\r\n")
}

func encodeArray(a []string) []byte {
	msg := fmt.Sprintf("*%d\r\n", len(a))
	for _, e := range a {
		msg = fmt.Sprintf("%s%s", msg, encodeBulk(e))
	}
	return []byte(msg)
}

func ReadNextRESP(b []byte) (n int, resp Response) {
	if len(b) == 0 {
		return 0, Response{}
	}

	resp.Type = Type(b[0])
	switch resp.Type {
	case Error, Status, Int, Bulk, Array:
	default:
		// Invalid Type
		return -1, Response{}
	}

	// Find next \r\n
	i := strings.Index(string(b), "\r\n")

	// Couldn't find it, so it's invalid
	if i == -1 {
		return -2, Response{}
	}

	i += 2 // move after \r\n

	resp.Raw = b[0:i]
	resp.Data = b[1 : i-2]

	if resp.Type == Int {
		err := parseInt(resp.Data)

		if err != nil {
			return 0, Response{}
		}

		return len(resp.Raw), resp
	}

	if resp.Type == Error {
		return len(resp.Raw), resp
	}

    if resp.Type == Status {
        return len(resp.Raw), resp
    }

	count, err := strconv.Atoi(resp.String())
	if err != nil {
		return 0, Response{}
	}

	if resp.Type == Bulk {
		err = parseBulk(count, i, b)

		if err != nil {
			return 0, Response{}
		}

		resp.Data = b[i : i+count]
		resp.Raw = b[0 : i+count+2]
		return len(resp.Raw), resp
	}

	// It's an array for sure
	var k int
	slice := b[i:]
	for j := 0; j < count; j++ {
		sn, subresp := ReadNextRESP(slice)

		if subresp.Type == 0 {
			return 0, Response{}
		}

		k += sn
		slice = slice[sn:]
	}

	resp.Data = b[i : i+k]
	resp.Raw = b[0 : i+k]
	resp.Count = count

	return len(resp.Raw), resp
}
