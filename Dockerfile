FROM golang:1.26.4-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/portfolio-api .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /app/database/migrations /app/public/uploads \
    && chown -R app:app /app

WORKDIR /app
COPY --from=build --chown=app:app /out/portfolio-api ./portfolio-api
COPY --from=build --chown=app:app /src/database/migrations ./database/migrations

ENV UPLOAD_DIR=/app/public/uploads
USER app
EXPOSE 10000
CMD ["./portfolio-api"]
