# Build stage
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/app
# Run stage
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /app/bin/server /usr/local/bin/server
WORKDIR /app
EXPOSE 8080
ENV DATABASE_URL=""
CMD ["/usr/local/bin/server"]