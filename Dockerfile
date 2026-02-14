FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache git make postgresql-client ca-certificates curl

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/air-verse/air@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest

COPY . .
RUN go mod tidy
RUN $GOPATH/bin/swag init
RUN go build -buildvcs=false -ldflags="-w -s" -o /main .

EXPOSE 8080

CMD ["air"]
