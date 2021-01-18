FROM golang:1.15.6-alpine as build
# ENV GO111MODULE=on
RUN mkdir -p /go/src/github.com/burnjake/loadbalancer
WORKDIR /go/src/github.com/burnjake/loadbalancer
COPY . .
RUN go mod download
RUN export CGO_ENABLED=0 GOOS=linux GOARCH=amd64 && go build -a -o /go/bin/loadbalancer cmd/loadbalancer/main.go

FROM scratch
COPY --from=build /go/bin/loadbalancer /go/bin/loadbalancer
EXPOSE 8090 8091 5353
ENTRYPOINT ["/go/bin/loadbalancer"]
