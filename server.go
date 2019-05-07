/*
Serve is a very simple static file server in go
Usage:
	-p="8100" or env.PORT:                 port to serve on
	-d="." or env.DIRECTORY:               the directory of static files to host
	-b="/" or env.BASE_URI:                base path of static files on the web
	-k="300" or env.KEEPALIVE:             http header keep-alive value
	-c="max-age=2800" or env.CACHECONTROL: http header Cache-controll: value
	-i="true" or env.DIRINDEX:             show directories index
	-l="true" or env.LOG:                  show requests logs
	-m="true" or env.METRICS:              generate / serve metrics
	-mp="9090" or env.METRICS_PORT:        serve metrics on port
*/
package main

import (
	"flag"
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
	dirIndex     bool
	log          bool
	metrics      bool
	metricsPort  string
}

func (cfg *config) getConfigs() {
	flagPort := flag.String("p", "8100", "port to serve on")
	flagDirectory := flag.String("d", ".", "the directory of static file to host")
	flagBaseURI := flag.String("b", "/", "the base path of static files")
	flagKeepAlive := flag.String("k", "", "http header keep-alive value")
	flagCacheControl := flag.String("c", "", "http header Cache-controll: value")
	flagDirIndex := flag.Bool("i", false, "show directories index")
	flagLog := flag.Bool("l", false, "show requests logs")
	flagMetrics := flag.Bool("m", false, "generate / serve metrics")
	flagMetricsPort := flag.String("mp", "9090", "serve metrics on port")

	flag.Parse()

	envPort := os.Getenv("PORT")
	if envPort != "" {
		cfg.port = envPort
	}
	if *flagPort != "8100" {
		cfg.port = *flagPort
	}

	envDirectory := os.Getenv("DIRECTORY")
	if envDirectory != "" {
		cfg.directory = envDirectory
	}
	if *flagDirectory != "." {
		cfg.directory = *flagDirectory
	}

	envBase := os.Getenv("BASE_URI")
	if envBase != "" {
		cfg.baseURI = envBase
	}
	if *flagBaseURI != "/" {
		cfg.baseURI = *flagBaseURI
	}

	envKeepAlive := os.Getenv("KEEPALIVE")
	if envKeepAlive != "" {
		cfg.keepAlive = envKeepAlive
	}
	if *flagKeepAlive != "" {
		cfg.keepAlive = *flagKeepAlive
	}

	envCacheControl := os.Getenv("CACHECONTROL")
	if envCacheControl != "" {
		cfg.cacheControl = envCacheControl
	}
	if *flagCacheControl != "" {
		cfg.cacheControl = *flagCacheControl
	}

	envDirIndex := os.Getenv("DIRINDEX")
	if envDirIndex != "" {
		cfg.dirIndex = true
	}
	if *flagDirIndex != false {
		cfg.dirIndex = *flagDirIndex
	}

	envLog := os.Getenv("LOG")
	if envLog != "" {
		cfg.log = true
	}
	if *flagLog != false {
		cfg.log = *flagLog
	}

	envMetrics := os.Getenv("METRICS")
	if envMetrics != "" {
		cfg.metrics = true
	}
	if *flagMetrics != false {
		cfg.metrics = *flagMetrics
	}

	envMetricsPort := os.Getenv("METRICS_PORT")
	if envMetricsPort != "" {
		cfg.metricsPort = envMetricsPort
	}
	if *flagMetricsPort != "9090" {
		cfg.metricsPort = *flagMetricsPort
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

func changeHeader(cfg config, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.keepAlive != "" {
			w.Header().Add("Keep-Alive", cfg.keepAlive)
		}
		if cfg.cacheControl != "" {
			w.Header().Add("Cache-Control", cfg.cacheControl)
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
	cfg := config{
		port:         "8100",
		directory:    ".",
		baseURI:      "/",
		keepAlive:    "",
		cacheControl: "",
		dirIndex:     false,
		log:          false,
		metrics:      false,
		metricsPort:  "9090"}
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
	headerChanger := changeHeader(cfg, loghandler)

	var h http.Handler
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
