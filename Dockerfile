FROM golang:latest

WORKDIR /middleware-bot

COPY . .

RUN go build ./cmd/middleware-services

CMD["./cmd/middleware-bot"]