package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

func main() {
	addr := flag.String("addr", "", "server address")
	flag.Parse()

	err := generateX509Certificate()
	if err != nil {
		fmt.Println(err)
		return
	}

	u, err := url.Parse(*addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	rp := httputil.NewSingleHostReverseProxy(u)
	rp.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // #nosec G402
		},
	}

	svr := &http.Server{
		Addr:    ":8080",
		Handler: rp,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, // // #nosec G402
		},
		ReadHeaderTimeout: 5 * time.Second,
	}

	err = svr.ListenAndServeTLS("cert.pem", "key.pem")
	if err != nil {
		fmt.Println(err)
		return
	}
}

func generateX509Certificate() error {
	// Generate the private key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating private key: %w", err)
	}

	// Use the private key to generate a PEM block.
	keyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	err = os.WriteFile("key.pem", keyPem, 0600)
	if err != nil {
		return fmt.Errorf("writing key.pem: %w", err)
	}

	// Generate the certificate.
	serial, err := rand.Int(rand.Reader, big.NewInt(2048))
	if err != nil {
		return fmt.Errorf("getting random number: %w", err)
	}
	tml := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "array-proxy",
			Organization: []string{"Dell"},
		},
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	// Use the certificate to generate a PEM block.
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	err = os.WriteFile("cert.pem", certPem, 0600)
	if err != nil {
		return fmt.Errorf("writing cert.pem: %w", err)
	}
	return nil
}
