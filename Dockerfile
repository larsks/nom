FROM docker.io/golang:alpine AS build

WORKDIR /app
RUN apk add make alpine-sdk gcc

# Cache depdenencies for faster rebuilds
COPY go.mod go.sum .
RUN go mod download

# Build nom. CGO_ENABLED is required due to our use of sqlite.
COPY . .
RUN CGO_ENABLED=1 make build

FROM docker.io/alpine:latest
COPY --from=build /app/nom /usr/local/bin/

WORKDIR /config/nom
ENV XDG_CONFIG_HOME=/config
COPY docker-config.yml ./config.yml
ENTRYPOINT ["nom"]
