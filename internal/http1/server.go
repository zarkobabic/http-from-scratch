package http1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
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

func handleParsingError(err error, conn net.Conn) error {
	code := StatusBadRequest
	var httpError *HTTPError
	if errors.As(err, &httpError) {
		code = httpError.Code
	}
	writeErr := writeErrorResponseBody(conn, code)
	if writeErr != nil {
		return fmt.Errorf("error response write failed: %w", writeErr)
	}
	return nil
}

func HandleConnection(conn net.Conn, mux http.Handler) error {
	reader := bufio.NewReader(conn)

	for {
		req, err := ParseRequest(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return handleParsingError(err, conn)
		}

		res := NewResponse(conn, req.Method)

		keepAlive := true
		if req.ProtoMinor == 0 && strings.ToLower(req.Header.Get("Connection")) != "keep-alive" {
			keepAlive = false
		} else if strings.ToLower(req.Header.Get("Connection")) == "close" {
			keepAlive = false
		}
		if !keepAlive {
			res.SetField("Connection", "close")
		}

		mux.ServeHTTP(res, req)

		if err := res.Finish(); err != nil {
			return fmt.Errorf("finish failed: %w", err)
		}
		_, err = io.Copy(io.Discard, req.Body)
		if err != nil {
			return fmt.Errorf("error while flushing %w", err)
		}
		err = req.Body.Close()
		if err != nil {
			return fmt.Errorf("error while body closure %w", err)
		}

		if !keepAlive {
			break
		}
	}
	return nil
}
