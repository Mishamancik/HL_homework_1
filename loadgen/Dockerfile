FROM golang:1.23.4-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/loadgen .

FROM alpine:3.20.3

RUN apk add --no-cache ca-certificates \
	&& adduser -D -H -u 10001 appuser

COPY --from=build /out/loadgen /usr/local/bin/loadgen

USER appuser
ENTRYPOINT ["/usr/local/bin/loadgen"]
