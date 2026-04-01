# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache ca-certificates
RUN adduser -D -g '' -u 10001 nonroot

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /server /server
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
