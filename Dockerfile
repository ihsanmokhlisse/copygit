FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /copygit ./cmd/copygit

FROM alpine:3.21

RUN apk add --no-cache git openssh-client ca-certificates && \
    adduser -D -h /home/copygit copygit

USER copygit
WORKDIR /home/copygit

COPY --from=builder /copygit /usr/local/bin/copygit

ENTRYPOINT ["copygit"]
