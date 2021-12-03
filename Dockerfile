FROM golang:1.16-stretch

ARG _ENV

ENV GO111MODULE auto
ENV GOPATH /go

WORKDIR /go/src/app
COPY . .

RUN go mod tidy
RUN go get -d -v ./...
RUN go install -v ./...

RUN ls -la

CMD ["/go/bin/websocket-service", "-configFile", "/go/src/app/config.json"]