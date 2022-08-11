FROM golang:1.19 AS build_base

## Usage:
## docker build . -t gokapi
## docker run -it -v gokapi-data:/app/data -v gokapi-config:/app/config -p 127.0.0.1:53842:53842 gokapi

RUN mkdir /compile
COPY go.mod /compile
RUN cd /compile && go mod download
  
COPY . /compile  

RUN cd /compile && go generate ./... && CGO_ENABLED=0 go build -ldflags="-s -w -X 'github.com/forceu/gokapi/internal/environment.IsDocker=true' -X 'github.com/forceu/gokapi/internal/environment.Builder=Project Docker File' -X 'github.com/forceu/gokapi/internal/environment.BuildTime=$(date)'" -o /compile/gokapi github.com/forceu/gokapi/cmd/gokapi

FROM alpine:3.13


RUN apk add ca-certificates && mkdir /app && touch /app/.isdocker
COPY --from=build_base /compile/gokapi /app/gokapi
WORKDIR /app

CMD ["/app/gokapi"]
