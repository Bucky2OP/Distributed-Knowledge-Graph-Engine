FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY main.go .

RUN go build -o graph-store main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/graph-store .

EXPOSE 8080

CMD ["./graph-store"]