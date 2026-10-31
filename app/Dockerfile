FROM golang:1.23.4-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app .

FROM alpine:3.20.3

RUN apk add --no-cache ca-certificates wget \
	&& adduser -D -H -u 10001 appuser

COPY --from=build /out/app /usr/local/bin/app

USER appuser
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]
