package http1

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
)

//TODO: implement custom writing
//Temporary tests with http standard library

type Response struct {
	conn net.Conn
	rec  *httptest.ResponseRecorder
}

func NewResponse(conn net.Conn) *Response {
	return &Response{
		conn: conn,
		rec:  httptest.NewRecorder(),
	}
}

func (r *Response) Header() http.Header {
	return r.rec.Header()
}

func (r *Response) Write(data []byte) (n int, err error) {
	return r.rec.Write(data)
}

func (r *Response) WriteHeader(statusCode int) {
	r.rec.WriteHeader(statusCode)
}

func (r *Response) Flush() error {
	res := r.rec.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("flush error", "error", err)
		}
	}(res.Body)
	return res.Write(r.conn)
}
