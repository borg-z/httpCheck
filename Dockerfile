FROM golang:1.12.1 AS builder
ADD https://github.com/golang/dep/releases/download/v0.5.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep
WORKDIR $GOPATH/src/github.com/borgkun/elastic-exporter
COPY Gopkg.toml Gopkg.lock main.go ./
RUN dep ensure --vendor-only
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o httpCheck

# final stage
FROM alpine
WORKDIR /app
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /go/src/github.com/borgkun/elastic-exporter/httpCheck /app/httpCheck
COPY  settings.yaml /app/settings.yaml
EXPOSE 8092
CMD /app/httpCheck
