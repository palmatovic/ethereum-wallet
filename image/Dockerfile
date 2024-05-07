FROM golang:latest AS ethereum-wallet-mods
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
FROM ethereum-wallet-mods AS ethereum-wallet-build
COPY cmd/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./main.go
FROM alpine:latest AS ethereum-wallet-run
RUN apk --no-cache add ca-certificates
RUN adduser appgo --disabled-password
WORKDIR /home/appgo
COPY --from=ethereum-wallet-build /app/main ./
RUN chown appgo:appgo -R .
USER appgo
CMD ["./main"]