FROM golang:1.26.2-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags='-s -w -linkmode external -extldflags "-static"' -trimpath -o gorkov .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /src/gorkov /app/gorkov
ENTRYPOINT ["/bin/sh", "-c", "exec /app/gorkov -token=\"$TOKEN\" -guildIDs=\"$GUILDS\" -db=\"$DB_PATH\""]
