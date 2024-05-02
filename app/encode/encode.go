package encode

import (
	"fmt"
	"strconv"
)

func EncodeBulkNoCrlf(str string) []byte {
	return []byte("$" + strconv.Itoa(len(str)) + "\r\n" + str)
}

func EncodeBulk(str string) []byte {
	return []byte("$" + strconv.Itoa(len(str)) + "\r\n" + str + "\r\n")
}

func EncodeArray(a []string) []byte {
	msg := fmt.Sprintf("*%d\r\n", len(a))
	for _, e := range a {
		msg = fmt.Sprintf("%s%s", msg, EncodeBulk(e))
	}
	return []byte(msg)
}
