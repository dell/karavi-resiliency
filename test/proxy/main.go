package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	addr := flag.String("addr", "", "server address")
	flag.Parse()

	u, err := url.Parse(*addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rp := httputil.NewSingleHostReverseProxy(u)
	rp.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	svr := &http.Server{
		Addr:    ":8080",
		Handler: rp,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	err = svr.ListenAndServeTLS("cert.pem", "key.pem")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
