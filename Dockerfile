FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache git make postgresql-client ca-certificates curl

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/air-verse/air@latest

COPY . .
RUN go build -buildvcs=false -ldflags="-w -s" -o /main .

EXPOSE 8080

CMD ["air"]
