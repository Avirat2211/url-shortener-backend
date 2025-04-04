FROM golang:1.24 AS builder  

WORKDIR /app

COPY . .

RUN go mod tidy  

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/main .
RUN chmod +x main  

EXPOSE 9808

CMD ["./main"]
