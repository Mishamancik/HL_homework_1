FROM golang:1.23.4-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/app \
	&& CGO_ENABLED=0 GOOS=linux go build -o /out/dashboard-gen ./cmd/dashboard-gen \
	&& CGO_ENABLED=0 GOOS=linux go build -o /out/loadgen ./cmd/loadgen

FROM alpine:3.20.3

RUN apk add --no-cache ca-certificates wget \
	&& adduser -D -H -u 10001 appuser

COPY --from=build /out/app /usr/local/bin/app
COPY --from=build /out/dashboard-gen /usr/local/bin/dashboard-gen
COPY --from=build /out/loadgen /usr/local/bin/loadgen

USER appuser
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]
