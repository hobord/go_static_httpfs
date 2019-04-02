/*
Serve is a very simple static file server in go
Usage:
	-p="8100" or env.PORT:      port to serve on
	-d="." or env.DIRECTORY:    the directory of static files to host
	-b="/" or env.BASE_URI:     base path of static files on the web
Navigating to http://localhost:8100
listing file.
*/
package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"os"
)

type justFilesFilesystem struct {
	Fs http.FileSystem
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.Fs.Open(name)

	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if stat.IsDir() {
		return nil, os.ErrNotExist
	}

	return f, nil
}

type etagResponseWriter struct {
	http.ResponseWriter
	buf  bytes.Buffer
	hash hash.Hash
	w    io.Writer
}

func (e *etagResponseWriter) Write(p []byte) (int, error) {
	return e.w.Write(p)
}

func getConfigs() (*string, *string, *string) {
	portFlag := flag.String("p", "8100", "port to serve on")
	directoryFlag := flag.String("d", ".", "the directory of static file to host")
	baseFlag := flag.String("b", "/", "the base path of static files")
	flag.Parse()

	if *portFlag == "8100" {
		envPort := os.Getenv("PORT")
		if envPort != "" {
			*portFlag = envPort
		}
	}

	if *directoryFlag == "." {
		envDirectory := os.Getenv("DIRECTORY")
		if envDirectory != "" {
			*directoryFlag = envDirectory
		}
	}

	if *baseFlag == "/" {
		envBase := os.Getenv("BASE_URI")
		if envBase != "" {
			*baseFlag = envBase
		}
	}

	return portFlag, directoryFlag, baseFlag
}

func changeHeaderThenServe(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Keep-Alive", "300")
		w.Header().Add("Cache-Control", "max-age=28800")

		// ETag
		ew := &etagResponseWriter{
			ResponseWriter: w,
			buf:            bytes.Buffer{},
			hash:           sha1.New(),
		}
		ew.w = io.MultiWriter(&ew.buf, ew.hash)

		sum := fmt.Sprintf("%x", ew.hash.Sum(nil))
		w.Header().Add("ETag", sum)

		if r.Header.Get("If-None-Match") == sum {
			w.WriteHeader(304)
		} else {
			_, err := ew.buf.WriteTo(w)
			if err != nil {
				fmt.Println("unable to write HTTP response")
			}
		}
		// ETag END

		h.ServeHTTP(w, r)
	}
}

func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(*r.URL)
		h.ServeHTTP(w, r)
	})
}

func main() {
	port, directory, baseURI := getConfigs()

	fs := justFilesFilesystem{http.Dir(*directory)}
	fileServer := http.FileServer(fs)
	stripPrefix := http.StripPrefix(*baseURI, fileServer)
	loghandler := logHandler(stripPrefix)
	headerChanger := changeHeaderThenServe(loghandler)

	http.Handle(*baseURI, headerChanger)

	// http.Handle(*baseURI, changeHeaderThenServe(logHandler((http.StripPrefix(*baseURI, http.FileServer(http.Dir(*directory)))))))

	log.Printf("Serving %s directory with %s basepath on HTTP port: %s\n", *directory, *baseURI, *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
