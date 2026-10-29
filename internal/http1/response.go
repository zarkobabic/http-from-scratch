package http1

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"slices"
)

type StatusCode int

const (
	StatusContinue                      StatusCode = 100
	StatusSwitchingProtocols            StatusCode = 101
	StatusEarlyHints                    StatusCode = 103
	StatusOK                            StatusCode = 200
	StatusCreated                       StatusCode = 201
	StatusAccepted                      StatusCode = 202
	StatusNonAuthoritativeInfo          StatusCode = 203
	StatusNoContent                     StatusCode = 204
	StatusResetContent                  StatusCode = 205
	StatusPartialContent                StatusCode = 206
	StatusMultipleChoices               StatusCode = 300
	StatusMovedPermanently              StatusCode = 301
	StatusFound                         StatusCode = 302
	StatusSeeOther                      StatusCode = 303
	StatusNotModified                   StatusCode = 304
	StatusTemporaryRedirect             StatusCode = 307
	StatusPermanentRedirect             StatusCode = 308
	StatusBadRequest                    StatusCode = 400
	StatusUnauthorized                  StatusCode = 401
	StatusPaymentRequired               StatusCode = 402
	StatusForbidden                     StatusCode = 403
	StatusNotFound                      StatusCode = 404
	StatusMethodNotAllowed              StatusCode = 405
	StatusNotAcceptable                 StatusCode = 406
	StatusProxyAuthRequired             StatusCode = 407
	StatusRequestTimeout                StatusCode = 408
	StatusConflict                      StatusCode = 409
	StatusGone                          StatusCode = 410
	StatusLengthRequired                StatusCode = 411
	StatusPreconditionFailed            StatusCode = 412
	StatusRequestEntityTooLarge         StatusCode = 413
	StatusRequestURITooLong             StatusCode = 414
	StatusUnsupportedMediaType          StatusCode = 415
	StatusRequestedRangeNotSatisfiable  StatusCode = 416
	StatusExpectationFailed             StatusCode = 417
	StatusMisdirectedRequest            StatusCode = 421
	StatusUnprocessableEntity           StatusCode = 422
	StatusTooEarly                      StatusCode = 425
	StatusUpgradeRequired               StatusCode = 426
	StatusPreconditionRequired          StatusCode = 428
	StatusTooManyRequests               StatusCode = 429
	StatusRequestHeaderFieldsTooLarge   StatusCode = 431
	StatusUnavailableForLegalReasons    StatusCode = 451
	StatusInternalServerError           StatusCode = 500
	StatusNotImplemented                StatusCode = 501
	StatusBadGateway                    StatusCode = 502
	StatusServiceUnavailable            StatusCode = 503
	StatusGatewayTimeout                StatusCode = 504
	StatusHTTPVersionNotSupported       StatusCode = 505
	StatusNetworkAuthenticationRequired StatusCode = 511
)

func statusPhrase(code StatusCode) string {
	switch code {
	case StatusContinue:
		return "Continue"
	case StatusSwitchingProtocols:
		return "Switching Protocols"
	case StatusEarlyHints:
		return "Early Hints"
	case StatusOK:
		return "OK"
	case StatusCreated:
		return "Created"
	case StatusAccepted:
		return "Accepted"
	case StatusNonAuthoritativeInfo:
		return "Non-Authoritative Information"
	case StatusNoContent:
		return "No Content"
	case StatusResetContent:
		return "Reset Content"
	case StatusPartialContent:
		return "Partial Content"
	case StatusMultipleChoices:
		return "Multiple Choices"
	case StatusMovedPermanently:
		return "Moved Permanently"
	case StatusFound:
		return "Found"
	case StatusSeeOther:
		return "See Other"
	case StatusNotModified:
		return "Not Modified"
	case StatusTemporaryRedirect:
		return "Temporary Redirect"
	case StatusPermanentRedirect:
		return "Permanent Redirect"
	case StatusBadRequest:
		return "Bad Request"
	case StatusUnauthorized:
		return "Unauthorized"
	case StatusPaymentRequired:
		return "Payment Required"
	case StatusForbidden:
		return "Forbidden"
	case StatusNotFound:
		return "Not Found"
	case StatusMethodNotAllowed:
		return "Method Not Allowed"
	case StatusNotAcceptable:
		return "Not Acceptable"
	case StatusProxyAuthRequired:
		return "Proxy Authentication Required"
	case StatusRequestTimeout:
		return "Request Timeout"
	case StatusConflict:
		return "Conflict"
	case StatusGone:
		return "Gone"
	case StatusLengthRequired:
		return "Length Required"
	case StatusPreconditionFailed:
		return "Precondition Failed"
	case StatusRequestEntityTooLarge:
		return "Content Too Large"
	case StatusRequestURITooLong:
		return "URI Too Long"
	case StatusUnsupportedMediaType:
		return "Unsupported Media Type"
	case StatusRequestedRangeNotSatisfiable:
		return "Range Not Satisfiable"
	case StatusExpectationFailed:
		return "Expectation Failed"
	case StatusMisdirectedRequest:
		return "Misdirected Request"
	case StatusUnprocessableEntity:
		return "Unprocessable Content"
	case StatusTooEarly:
		return "Too Early"
	case StatusUpgradeRequired:
		return "Upgrade Required"
	case StatusPreconditionRequired:
		return "Precondition Required"
	case StatusTooManyRequests:
		return "Too Many Requests"
	case StatusRequestHeaderFieldsTooLarge:
		return "Request Header Fields Too Large"
	case StatusUnavailableForLegalReasons:
		return "Unavailable For Legal Reasons"
	case StatusInternalServerError:
		return "Internal Server Error"
	case StatusNotImplemented:
		return "Not Implemented"
	case StatusBadGateway:
		return "Bad Gateway"
	case StatusServiceUnavailable:
		return "Service Unavailable"
	case StatusGatewayTimeout:
		return "Gateway Timeout"
	case StatusHTTPVersionNotSupported:
		return "HTTP Version Not Supported"
	case StatusNetworkAuthenticationRequired:
		return "Network Authentication Required"
	default:
		return ""
	}
}

type Response struct {
	HTTPVersion  string
	StatusCode   StatusCode
	ReasonPhrase string

	FieldLine map[string][]string

	reqMethod   string
	conn        net.Conn
	wroteHeader bool
	isChunked   bool
}

func (r *Response) Header() http.Header {
	return r.FieldLine
}

func NewResponse(conn net.Conn, reqMethod string) *Response {
	return &Response{
		reqMethod:   reqMethod,
		HTTPVersion: "HTTP/1.1",
		StatusCode:  StatusOK,
		FieldLine:   make(map[string][]string),
		conn:        conn,
	}
}

func writeErrorResponseBody(w io.Writer, code StatusCode) error {
	_, err := w.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\nConnection: close\r\n\r\n", code, statusPhrase(code))))
	if err != nil {
		return fmt.Errorf("writing of error response failed: %w", err)
	}
	return nil
}

func writeErrorResponse(w http.ResponseWriter, code StatusCode) error {
	w.WriteHeader(int(code))
	return writeErrorResponseBody(w, code)
}

func (r *Response) SetField(key, value string) {
	r.FieldLine[key] = []string{value}
}

//func (r *Response) AddField(key, value string) {
//	r.FieldLine[key] = append(r.FieldLine[key], value)
//}

func (r *Response) determineBodyFraming(statusCode StatusCode) {
	bodyForbiddenByStatus := (statusCode >= 100 && statusCode < 200) || statusCode == StatusNoContent || statusCode == StatusNotModified
	isConnectTunnel := r.reqMethod == "CONNECT" && statusCode >= 200 && statusCode < 300

	if bodyForbiddenByStatus || isConnectTunnel {
		delete(r.FieldLine, "Content-Length")
		delete(r.FieldLine, "Transfer-Encoding")
		r.isChunked = false
	} else {
		_, hasContentLength := r.FieldLine["Content-Length"]
		transferEncodingValues, hasTransferEncoding := r.FieldLine["Transfer-Encoding"]

		if hasTransferEncoding && hasContentLength {
			delete(r.FieldLine, "Content-Length")
			hasContentLength = false
		}

		if !hasContentLength && !hasTransferEncoding {
			r.isChunked = true
			r.SetField("Transfer-Encoding", "chunked")
		} else if hasTransferEncoding {
			if slices.Contains(transferEncodingValues, "chunked") {
				r.isChunked = true
			}
		}
	}
}

func (r *Response) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}
	r.StatusCode = StatusCode(statusCode)
	r.wroteHeader = true
	r.ReasonPhrase = statusPhrase(StatusCode(statusCode))

	r.determineBodyFraming(StatusCode(statusCode))

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s %d %s\r\n", r.HTTPVersion, r.StatusCode, r.ReasonPhrase))

	for key, values := range r.FieldLine {
		for _, v := range values {
			buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, v))
		}
	}
	buf.WriteString("\r\n")

	_, err := r.conn.Write(buf.Bytes())
	if err != nil {
		slog.Error("writing of the headers failed", "error", err)
		return
	}
}

func (r *Response) Write(data []byte) (n int, err error) {
	if !r.wroteHeader {
		r.WriteHeader(200)
	}

	statusCode := r.StatusCode
	bodyForbidden := (statusCode >= 100 && statusCode < 200) || statusCode == StatusNoContent || statusCode == StatusNotModified
	isConnectTunnel := r.reqMethod == "CONNECT" && statusCode >= 200 && statusCode < 300
	if bodyForbidden || isConnectTunnel || len(data) == 0 {
		return 0, nil
	}

	if r.isChunked {
		chunkHeader := fmt.Sprintf("%x\r\n", len(data))
		_, err := r.conn.Write([]byte(chunkHeader))
		if err != nil {
			return 0, fmt.Errorf("writing to connection failed: %w", err)
		}
		n, err = r.conn.Write(data)
		if err != nil {
			return 0, fmt.Errorf("writing to connection failed: %w", err)
		}
		_, err = r.conn.Write([]byte("\r\n"))
		if err != nil {
			return 0, fmt.Errorf("writing to connection failed: %w", err)
		}
		return n, err
	}

	n, err = r.conn.Write(data)
	if err != nil {
		return 0, fmt.Errorf("writing to connection failed: %w", err)
	}
	return n, err
}

func (r *Response) Finish() error {
	if !r.wroteHeader {
		r.WriteHeader(int(StatusOK))
	}
	if r.isChunked {
		_, err := r.conn.Write([]byte("0\r\n\r\n"))
		return err
	}
	return nil
}
