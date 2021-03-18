FROM golang:1.16 AS build_base

## Usage:
## docker build . -t gokapi
## docker run -it -v gokapi-data:/app/data -v gokapi-config:/app/config -p 127.0.0.1:53842:53842 gokapi

RUN mkdir /compile
COPY go.mod /compile
RUN cd /compile && go mod download
  
COPY . /compile  

RUN cd /compile && CGO_ENABLED=0 go build -ldflags="-s -w -X 'main.IS_DOCKER=true' -X 'main.BUILDER=Project Docker File' -X 'main.BUILD_TIME=$(date)'" -o /compile/gokapi

FROM alpine:3.13


RUN apk add ca-certificates && mkdir /app && touch /app/.isdocker
COPY --from=build_base /compile/gokapi /app/gokapi
WORKDIR /app

CMD ["/app/gokapi"]


