FROM golang:1.24.0-alpine AS build_base

## Usage:
## docker build . -t gokapi
## docker run -it -v gokapi-data:/app/data -v gokapi-config:/app/config -p 127.0.0.1:53842:53842 gokapi

RUN mkdir /compile
COPY go.mod /compile
RUN cd /compile && go mod download
  
COPY . /compile  

RUN cd /compile && go generate ./... && CGO_ENABLED=0 go build -ldflags="-s -w -X 'github.com/forceu/gokapi/internal/environment.IsDocker=true' -X 'github.com/forceu/gokapi/internal/environment.Builder=Project Docker File' -X 'github.com/forceu/gokapi/internal/environment.BuildTime=$(date)'" -o /compile/gokapi github.com/forceu/gokapi/cmd/gokapi

FROM alpine:3.19


RUN addgroup -S gokapi && adduser -S gokapi -G gokapi
RUN apk update && apk add --no-cache su-exec tini ca-certificates curl tzdata && \
	 mkdir /app && touch /app/.isdocker

COPY dockerentry.sh /app/run.sh


COPY --from=build_base /compile/gokapi /app/gokapi
WORKDIR /app

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/run.sh"]
HEALTHCHECK --interval=10s --timeout=5s --retries=3 CMD curl --fail http://127.0.0.1:53842 || exit 1
