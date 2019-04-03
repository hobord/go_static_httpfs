/*
Serve is a very simple static file server in go
Usage:
	-p="8100" or env.PORT:      port to serve on
	-d="." or env.DIRECTORY:    the directory of static files to host
	-b="/" or env.BASE_URI:     base path of static files on the web

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

type config struct {
	port      string
	directory string
	baseURI   string
	keepAlive string
	maxAge    string
	dirIndex  bool
	log       bool
}

func (cfg *config) getConfigs() {
	flag.StringVar(&cfg.port, "p", "8100", "port to serve on")
	flag.StringVar(&cfg.directory, "d", ".", "the directory of static file to host")
	flag.StringVar(&cfg.baseURI, "b", "/", "the base path of static files")
	flag.StringVar(&cfg.keepAlive, "k", "", "Header keep-alive value")
	flag.StringVar(&cfg.maxAge, "m", "", "Cache-controll max-age value")
	flag.BoolVar(&cfg.dirIndex, "i", false, "Show directories index")
	flag.BoolVar(&cfg.log, "l", false, "Show request logs")
	flag.Parse()

	if cfg.port == "8100" {
		envPort := os.Getenv("PORT")
		if envPort != "" {
			cfg.port = envPort
		}
	}

	if cfg.directory == "." {
		envDirectory := os.Getenv("DIRECTORY")
		if envDirectory != "" {
			cfg.directory = envDirectory
		}
	}

	if cfg.baseURI == "/" {
		envBase := os.Getenv("BASE_URI")
		if envBase != "" {
			cfg.baseURI = envBase
		}
	}

	if cfg.keepAlive == "" {
		envKeepAlive := os.Getenv("KEEPALIVE")
		if envKeepAlive != "" {
			cfg.baseURI = envKeepAlive
		}
	}

	if cfg.maxAge == "" {
		envMaxAge := os.Getenv("MAXAGE")
		if envMaxAge != "" {
			cfg.maxAge = envMaxAge
		}
	}

	if cfg.dirIndex == false {
		envDirIndex := os.Getenv("DIRINDEX")
		if envDirIndex != "" {
			cfg.dirIndex = true
		}
	}

	if cfg.log == false {
		envLog := os.Getenv("LOG")
		if envLog != "" {
			cfg.log = true
		}
	}
}

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

func changeHeaderThenServe(cfg config, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.keepAlive != "" {
			w.Header().Add("Keep-Alive", cfg.keepAlive)
		}
		if cfg.maxAge != "" {
			w.Header().Add("Cache-Control", "max-age="+cfg.maxAge)
		}

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

func logHandler(cfg config, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.log {
			log.Printf("%s", r.URL)
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	cfg := config{}
	cfg.getConfigs()

	var fs http.FileSystem
	if cfg.dirIndex {
		fs = http.Dir(cfg.directory)
	} else {
		fs = justFilesFilesystem{http.Dir(cfg.directory)}
	}

	fileServer := http.FileServer(fs)
	stripPrefix := http.StripPrefix(cfg.baseURI, fileServer)
	loghandler := logHandler(cfg, stripPrefix)
	headerChanger := changeHeaderThenServe(cfg, loghandler)

	http.Handle(cfg.baseURI, headerChanger)

	log.Printf("Serving %s directory with %s basepath on HTTP port: %s\n", cfg.directory, cfg.baseURI, cfg.port)
	log.Fatal(http.ListenAndServe(":"+cfg.port, nil))
}
