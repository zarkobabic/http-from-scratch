package http1

import (
	"bufio"
	"net/http"
)

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {
	//TODO: implement custom parsing
	//Temporary test with http standard library
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}
	return req, nil
}
