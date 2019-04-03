
# build stage
FROM golang:1.11-alpine AS build-env
RUN apk add --no-cache git mercurial

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go build -o server

# final stage
FROM alpine
WORKDIR /app/public/
COPY --from=build-env /go/src/app/server /app/

EXPOSE 80 443 8080 8100 9090 3000
VOLUME ["/app/public/"]
ENTRYPOINT /app/server
