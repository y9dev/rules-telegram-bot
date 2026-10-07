FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bot ./cmd/bot

FROM alpine:3.20
WORKDIR /app
COPY --from=build /bot /app/bot
COPY rules.json /app/rules.json
ENV RULES_PATH=/app/rules.json
CMD ["/app/bot"]
