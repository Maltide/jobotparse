FROM golang:1.25.5-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out && go build -o /out/main .
FROM alpine:latest
WORKDIR /app

# Runtime dependencies:
# - ca-certificates: required for HTTPS requests (e.g., SuperJob API)
RUN apk add --no-cache poppler-utils ca-certificates && update-ca-certificates

COPY --from=build /out/main ./main
COPY --from=build /app/static ./static

EXPOSE 8080
CMD ["./main"]