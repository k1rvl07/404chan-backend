FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache git make postgresql-client ca-certificates curl

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/pressly/goose/v3/cmd/goose@latest && \
    go install github.com/air-verse/air@latest

COPY . .
RUN go build -ldflags="-w -s" -o /main .

CMD ["sleep", "infinity"]
