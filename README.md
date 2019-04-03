# Static http file server

Serve is a very simple static file server in go

## Build

linux || mac
```
go build -o server server.go
```


windows
```
go build -o server.exe server.go
```


## Usage
```
	-p="8100" or env.PORT:                 port to serve on
	-d="." or env.DIRECTORY:               the directory of static files to host
	-b="/" or env.BASE_URI:                base uri of static files on the web
	-k="300" or env.KEEPALIVE:             http header keep-alive value
	-c="max-age=2800" or env.CACHECONTROL: http header Cache-controll: value
	-i="true" or env.DIRINDEX:             show directories index
	-l="true" or env.LOG:                  show requests logs
```
