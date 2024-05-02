package client

import (
	"net"
	"strconv"
)

type Client struct {
	conn net.Conn
}

func EncodeBulk(str string) []byte {
	return []byte("$" + strconv.Itoa(len(str)) + "\r\n" + str + "\r\n")
}

func (c *Client) SendEncodeBulkString(str string) error {
	return c.Send(EncodeBulk(str))
}

func (c *Client) Send(buf []byte) error {
	_, err := c.conn.Write(buf)
	return err
}

func (c *Client) SendSuccess() error {
    return c.Send([]byte("+OK\r\n"))
}
