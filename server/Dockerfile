FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY proxy/ ./proxy/
RUN CGO_ENABLED=0 GOOS=linux go build -o wg0proxy .

FROM alpine:latest

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/wg0proxy .

EXPOSE 1081 8080

ENTRYPOINT ["./wg0proxy"]
CMD ["serve"]
