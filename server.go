/*
Serve is a very simple static file server in go
Usage:
	-p="8100" or env.PORT:                 port to serve on
	-d="." or env.DIRECTORY:               the directory of static files to host
	-b="/" or env.BASE_URI:                base path of static files on the web
	-k="300" or env.KEEPALIVE:             http header keep-alive value
	-c="max-age=2800" or env.CACHECONTROL: http header Cache-controll: value
	-e="true" or env.ETAG:                 calculate and add etag from file
	-i="true" or env.DIRINDEX:             show directories index
	-l="true" or env.LOG:                  show requests logs
	-m="true" or env.METRICS:              generate / serve metrics
	-mp="9090" or env.METRICS_PORT:        serve metrics on port
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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
)

type config struct {
	port         string
	directory    string
	baseURI      string
	keepAlive    string
	cacheControl string
	etag         bool
	dirIndex     bool
	log          bool
	metrics      bool
	metricsPort  string
}

func (cfg *config) getConfigs() {
	flag.StringVar(&cfg.port, "p", "8100", "port to serve on")
	flag.StringVar(&cfg.directory, "d", ".", "the directory of static file to host")
	flag.StringVar(&cfg.baseURI, "b", "/", "the base path of static files")
	flag.StringVar(&cfg.keepAlive, "k", "", "http header keep-alive value")
	flag.StringVar(&cfg.cacheControl, "c", "", "http header Cache-controll: value")
	flag.BoolVar(&cfg.etag, "e", false, "calculate and add etag from file")
	flag.BoolVar(&cfg.dirIndex, "i", false, "show directories index")
	flag.BoolVar(&cfg.log, "l", false, "show requests logs")
	flag.BoolVar(&cfg.metrics, "m", false, "generate / serve metrics")
	flag.StringVar(&cfg.metricsPort, "mp", "9090", "serve metrics on port")
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

	if cfg.cacheControl == "" {
		envCacheControl := os.Getenv("CACHECONTROL")
		if envCacheControl != "" {
			cfg.cacheControl = envCacheControl
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

	if cfg.etag == false {
		envEtag := os.Getenv("ETAG")
		if envEtag != "" {
			cfg.etag = true
		}
	}

	if cfg.metrics == false {
		envMetrics := os.Getenv("METRICS")
		if envMetrics != "" {
			cfg.metrics = true
		}
	}

	if cfg.metricsPort == "9090" {
		envMetricsPort := os.Getenv("METRICS_PORT")
		if envMetricsPort != "" {
			cfg.metricsPort = envMetricsPort
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
		if cfg.cacheControl != "" {
			w.Header().Add("Cache-Control", cfg.cacheControl)
		}

		if cfg.etag {
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

		}

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

	var h http.Handler = nil
	if cfg.metrics {
		mdlw := middleware.New(middleware.Config{
			Recorder: metrics.NewRecorder(metrics.Config{}),
		})

		h = mdlw.Handler(cfg.baseURI, headerChanger)
		log.Printf("serving metrics at: :%s", cfg.metricsPort)
		go http.ListenAndServe(":"+cfg.metricsPort, promhttp.Handler())
	} else {
		http.Handle(cfg.baseURI, headerChanger)
	}

	log.Printf("Serving %s directory with %s basepath on HTTP port: %s\n", cfg.directory, cfg.baseURI, cfg.port)
	log.Fatal(http.ListenAndServe(":"+cfg.port, h))
}
