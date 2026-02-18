FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /reviewforge .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /reviewforge /reviewforge

ENTRYPOINT ["/reviewforge"]
