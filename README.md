# Static http file server

Serve is a very simple static file server in go.

You can map/share a directory with specific base uri, and you can show/hide the directory indexes.

It is support basic cache control headers.
-  keepalive
-  cace-controll
-  Last-Modified

It is support serve prometius metrics also.

## Docker
image: hobord/go_static_httpfs
```
docker run -p 8100:8100 -v ./your_public:/app/public -e DIRECTORY=/app/public -e LOG=true hobord/go_static_httpfs
```

## Build

linux || mac
```
go get ./...
go build -o server server.go
```


windows
```
go get ./...
go build -o server.exe server.go
```


## Usage
```
-p="8100"         or env.PORT:         port to serve on
-d="."            or env.DIRECTORY:    the directory of static files to host
-b="/"            or env.BASE_URI:     base uri of static files on the web
-k="300"          or env.KEEPALIVE:    http header keep-alive value
-c="max-age=2800" or env.CACHECONTROL: http header Cache-controll: value
-i="true"         or env.DIRINDEX:     show directories index
-l="true"         or env.LOG:          show requests logs
-m="true"         or env.METRICS:      generate / serve metrics
-mp="9090"        or env.METRICS_PORT: serve metrics on port
```

