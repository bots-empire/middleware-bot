FROM golang:latest

WORKDIR /middleware-bot

COPY . .

RUN go build ./cmd/middleware-bot

CMD ["./middleware-bot"]