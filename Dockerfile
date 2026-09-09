FROM golang:1.27-alpine3.24

RUN addgroup -g 1000 go && adduser -h /home/go -G go -D -u 1000 go