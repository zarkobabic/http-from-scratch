package http1

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

func ListenAndServe(addr string, handler http.Handler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {

		return fmt.Errorf("listen: %w", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			slog.Error("listening socket closure failure", "error", err)
		}
	}(listener)

	if handler == nil {
		handler = http.DefaultServeMux
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}

		go func() {
			defer func() {
				err := conn.Close()
				if err != nil {
					slog.Error("connection closure failed", "error", err)
				}
			}()
			err := HandleConnection(conn, handler)
			if err != nil {
				slog.Error("connection handling failed", "error", err)
			}
		}()
	}
}

func HandleConnection(conn net.Conn, mux http.Handler) error {

	reader := bufio.NewReader(conn)
	req, err := ParseRequest(reader)
	if err != nil {

		return fmt.Errorf("failed to parse request: %w", err)
	}

	res := NewResponse(conn)
	mux.ServeHTTP(res, req)

	//Testing purposes
	if err := res.Flush(); err != nil {
		return fmt.Errorf("failed to write response to connection: %w", err)
	}

	return nil
}
