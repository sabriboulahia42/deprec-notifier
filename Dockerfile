FROM golang:1.20-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o notifier main.go
RUN CGO_ENABLED=0 go build -o envcrypt encrypt_decrypt.go

FROM alpine:3.18
WORKDIR /app
COPY --from=build /app/notifier /app/notifier
COPY --from=build /app/envcrypt /app/envcrypt
COPY templates /app/templates
COPY config.yaml /app/config.yaml
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh /app/envcrypt /app/notifier
VOLUME /app/data
ENV TZ=UTC
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/notifier"]
