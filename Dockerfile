FROM golang:1.11-alpine

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 80 443 8080 8100 9090 3000
CMD ["go_static_httpfs"]