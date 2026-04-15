# build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o main .

# run stage
FROM alpine:latest

WORKDIR /app

RUN adduser -D appuser
USER appuser

COPY --from=builder /app/main .

COPY --from=builder /app/web ./web

EXPOSE 8080

ENV PORT=8080

HEALTHCHECK CMD wget --spider http://localhost:8080/health || exit 1

CMD ["./main"]