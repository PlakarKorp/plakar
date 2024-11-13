FROM golang:1.23 AS builder

WORKDIR /go/src

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .
RUN go install ./...

FROM ubuntu

COPY --from=builder /go/bin/plakar /usr/local/bin/plakar

CMD ["/usr/local/bin/plakar"]