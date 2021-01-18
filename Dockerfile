FROM golang:1.15.6-alpine as build

ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -a -o cmd/loadbalancer/main.go
