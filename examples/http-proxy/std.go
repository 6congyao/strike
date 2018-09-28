package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var cmd Cmd
var srv http.Server

type handle struct {
	metadataServer string
}

type Cmd struct {
	bind           string
	metadataServer string
}

func parseCmd() Cmd {
	var cmd Cmd
	flag.StringVar(&cmd.bind, "l", "0.0.0.0:9090", "listen on ip:port")
	flag.StringVar(&cmd.metadataServer, "r", "127.0.0.1:9611", "metadata server addr")
	flag.Parse()
	return cmd
}

func StartServer(bind string, remote string) {
	log.Printf("Listening on %s, forwarding to %s", bind, remote)
	h := &handle{metadataServer: remote}
	srv.Addr = bind
	srv.Handler = h
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln("ListenAndServe: ", err)
	}
}

func (this *handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote, err := url.Parse("http://" + this.metadataServer)
	if err != nil {
		log.Fatal("parse metadata server url error:", err)
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(w, r)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal("Main Recover: %v, try restart.", r)
			time.Sleep(time.Duration(1000) * time.Millisecond)
			main()
		}
	}()

	cmd = parseCmd()
	StartServer(cmd.bind, cmd.metadataServer)
}
