package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/choonkeat/httppub/broadcast"
)

func main() {
	addr := flag.String("addr", ":3000", "address to listen at")
	timeout := flag.Duration("timeout", 30*time.Second, "maximum target request duration; maximum delay for shutdown of main app")
	targets := flag.String("targets", "http://localhost:5000,http://127.0.0.1:5001/pre/fix", "comma separated target urls; first url is primary")
	flag.Parse()

	h := &broadcast.Server{
		RequestTimeout: *timeout,
	}
	for _, target := range strings.Split(*targets, ",") {
		u, err := url.Parse(target)
		if err != nil {
			log.Fatalln(target, ":", err.Error())
		}
		h.Targets = append(h.Targets, *u)
	}

	server := http.Server{
		Addr:    *addr,
		Handler: h,
	}

	var serverErr error
	defer func() {
		log.Printf("[server] Stopped listening; cleaning up could take up to %s...", timeout)
		h.CleanupWait()
		log.Printf("[server] Graceful exit: %s", serverErr.Error())
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		<-sigs
		server.Close() // stops `ListenAndServe`
	}()

	log.Printf("[server] Listening at %s for %s", *addr, *targets)
	serverErr = server.ListenAndServe()
}
