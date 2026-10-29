package http1

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const MaxLineSize = 8000
const MaxNumHeaders = 1000
const MaxSingleHeaderSize = 8192

type HTTPError struct {
	Code StatusCode
	Err  error
}

type RequestLine struct {
	method  string
	target  string
	version string
	major   int
	minor   int
}

func (e *HTTPError) Error() string { return e.Err.Error() }

func isValidToken(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b <= 32 || b >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={} %", rune(b)) {
			return false
		}
	}
	return true
}

func readLineStrict(r *bufio.Reader, limit int) (string, error) {
	var line []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(line) == 0 {
					return "", io.EOF
				}
				return "", io.ErrUnexpectedEOF
			}
			return "", err
		}
		line = append(line, b)
		if len(line) > limit {
			return "", &HTTPError{Code: StatusRequestHeaderFieldsTooLarge, Err: errors.New("line too long")}
		}
		if b == '\n' {
			break
		}
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return "", &HTTPError{Code: StatusBadRequest, Err: errors.New("bare LF not allowed, missing CR")}
	}
	return string(line[:len(line)-2]), nil
}

func ParseRequestLine(reader *bufio.Reader) (*RequestLine, error) {
	reqLine, err := readLineStrict(reader, MaxLineSize)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		var httpError *HTTPError
		if errors.As(err, &httpError) {
			if httpError.Code == StatusRequestHeaderFieldsTooLarge {
				httpError.Code = StatusRequestURITooLong
			}
			return nil, httpError
		}
		return nil, &HTTPError{Code: StatusBadRequest, Err: err}
	}
	if reqLine == "" {
		return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("empty request line")}
	}
	parts := strings.Split(reqLine, " ")
	if len(parts) != 3 {
		return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("malformed request line")}
	}
	method, target, version := parts[0], parts[1], parts[2]

	if !isValidToken(method) {
		return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid method token")}
	}

	if !strings.HasPrefix(version, "HTTP/") {
		return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid version prefix")}
	}
	versionParts := strings.Split(version[5:], ".")
	if len(versionParts) != 2 {
		return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid version format")}
	}
	major, errMaj := strconv.Atoi(versionParts[0])
	minor, errMin := strconv.Atoi(versionParts[1])
	if errMaj != nil || errMin != nil {
		return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid version numbers")}
	}
	if major != 1 || (minor != 0 && minor != 1) {
		return nil, &HTTPError{Code: StatusHTTPVersionNotSupported, Err: errors.New("HTTP version not supported")}
	}
	return &RequestLine{method: method, target: target, version: version, major: major, minor: minor}, nil
}

func ParseHeaders(reader *bufio.Reader) (http.Header, error) {
	header := make(http.Header)
	for i := 0; ; i++ {
		if i > MaxNumHeaders {
			return nil, &HTTPError{Code: StatusRequestHeaderFieldsTooLarge, Err: errors.New("too many headers")}
		}
		line, err := readLineStrict(reader, MaxSingleHeaderSize)
		if err != nil {
			var httpError *HTTPError
			if errors.As(err, &httpError) {
				return nil, httpError
			}
			return nil, &HTTPError{Code: StatusBadRequest, Err: err}
		}
		if line == "" {
			break
		}

		colonIndex := strings.IndexByte(line, ':')
		if colonIndex <= 0 {
			return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid header format")}
		}
		key := line[:colonIndex]
		if key[len(key)-1] == ' ' || key[len(key)-1] == '\t' {
			return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("whitespace before colon")}
		}
		if !isValidToken(key) {
			return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid header key")}
		}

		value := strings.Trim(line[colonIndex+1:], " \t")
		for j := 0; j < len(value); j++ {
			if value[j] < 32 || value[j] == 127 {
				return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid character in header value")}
			}
		}
		header.Add(key, value)
	}
	return header, nil
}

func validateBodyFraming(reader *bufio.Reader, req *http.Request) (request *http.Request, err error) {
	transferEncoding := req.Header.Values("Transfer-Encoding")
	contentLengthValues := req.Header.Values("Content-Length")

	isChunked := false
	for _, v := range transferEncoding {
		for p := range strings.SplitSeq(v, ",") {
			if strings.ToLower(strings.TrimSpace(p)) == "chunked" {
				isChunked = true
			}
		}
	}
	var bodyData []byte

	if isChunked {
		bodyData, err = parseChunks(reader, req)
		if err != nil {
			return nil, err
		}
	} else if len(contentLengthValues) > 0 {
		bodyData, err = parseFixedLengthBody(reader, contentLengthValues)
		if err != nil {
			return nil, err
		}
	}

	req.Body = io.NopCloser(bytes.NewReader(bodyData))
	request = req
	return request, nil
}

func parseChunks(reader *bufio.Reader, req *http.Request) (bodyData []byte, err error) {
	req.Header.Del("Content-Length")
	for {
		line, err := readLineStrict(reader, 1024)
		if err != nil {
			return nil, &HTTPError{Code: StatusBadRequest, Err: err}
		}

		chunkExtensions := strings.IndexByte(line, ';')
		if chunkExtensions >= 0 {
			line = line[:chunkExtensions]
		}

		size, err := strconv.ParseInt(strings.TrimSpace(line), 16, 64)
		if err != nil || size < 0 {
			return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid chunk size")}
		}

		//Reading chunk trailers from the last zero chunk until the next line
		if size == 0 {
			for {
				trailersLine, err := readLineStrict(reader, 8192)
				if err != nil {
					return nil, &HTTPError{Code: StatusBadRequest, Err: err}
				}
				if trailersLine == "" {
					break
				}
			}
			break
		}

		chunk := make([]byte, size)
		if _, err := io.ReadFull(reader, chunk); err != nil {
			return nil, &HTTPError{Code: StatusBadRequest, Err: err}
		}
		bodyData = append(bodyData, chunk...)
		crlf := make([]byte, 2)
		if _, err := io.ReadFull(reader, crlf); err != nil || crlf[0] != '\r' || crlf[1] != '\n' {
			return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid chunk end")}
		}
	}
	return bodyData, nil
}

func parseFixedLengthBody(reader *bufio.Reader, contentLengthValues []string) (bodyData []byte, err error) {
	val := contentLengthValues[0]
	for _, v := range contentLengthValues[1:] {
		if v != val {
			return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("conflicting Content-Length")}
		}
	}
	contentLength, err := strconv.ParseInt(val, 10, 64)
	if err != nil || contentLength < 0 {
		return nil, &HTTPError{Code: StatusBadRequest, Err: errors.New("invalid Content-Length")}
	}

	bodyData = make([]byte, contentLength)
	if _, err := io.ReadFull(reader, bodyData); err != nil {
		return nil, &HTTPError{Code: StatusBadRequest, Err: err}
	}
	return bodyData, nil
}

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {
	requestLine, err := ParseRequestLine(reader)
	if err != nil {
		return nil, err
	}

	header, err := ParseHeaders(reader)
	if err != nil {
		return nil, err
	}

	reqURL, err := url.ParseRequestURI(requestLine.target)
	if err != nil {
		reqURL = &url.URL{Path: requestLine.target}
	}

	req := &http.Request{
		Method:     requestLine.method,
		URL:        reqURL,
		Proto:      requestLine.version,
		ProtoMajor: requestLine.major,
		ProtoMinor: requestLine.minor,
		Header:     header,
		RequestURI: requestLine.target,
	}

	return validateBodyFraming(reader, req)
}
