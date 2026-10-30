FROM golang:tip-alpine3.24@sha256:2cbff72a3aa3b69bc6f869a0385b886cb903195a8108a1ac9dc36ca607b26bf5 AS build
WORKDIR /http
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /http/server ./cmd/httpserver

FROM scratch
WORKDIR /http
COPY --from=build /http/server ./server
EXPOSE 8080
CMD ["./server"]
