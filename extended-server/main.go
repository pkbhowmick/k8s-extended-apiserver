package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/certstore"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/server"
	"k8s.io/client-go/util/cert"
)

func main() {
	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs, "../crt")
	handleErr(err)

	err = store.NewCA("database")
	handleErr(err)

	serverCert, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.2")},
	})
	handleErr(err)

	err = store.Write("tls", serverCert, serverKey)
	handleErr(err)

	clientCert, clientKey, err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"bob"},
	})
	handleErr(err)

	err = store.Write("bob", clientCert, clientKey)
	handleErr(err)

	// *************************************
	apiserverStore, err := certstore.NewCertStore(fs, "../crt")
	handleErr(err)

	err = apiserverStore.LoadCA("apiserver")
	handleErr(err)

	// ***************************************
	rhCACertPool := x509.NewCertPool()
	rhStore, err := certstore.NewCertStore(fs, "../crt")
	handleErr(err)

	err = rhStore.LoadCA("requestheader")
	handleErr(err)
	rhCACertPool.AppendCertsFromPEM(rhStore.CACertBytes())

	// ****************************************

	cfg := server.Config{
		Address:     "127.0.0.2:8443",
		CACertFiles: []string{},
		CertFile:    store.CertFile("tls"),
		KeyFile:     store.KeyFile("tls"),
	}
	cfg.CACertFiles = append(cfg.CACertFiles, apiserverStore.CertFile("ca"))
	cfg.CACertFiles = append(cfg.CACertFiles, rhStore.CertFile("ca"))
	srv := server.NewGenericServer(cfg)

	r := mux.NewRouter()
	r.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(res, "Extended Server is OK")
	})
	r.HandleFunc("/database/{resource}", func(res http.ResponseWriter, req *http.Request) {
		user := "system:anonymous"
		src := "-"
		if len(req.TLS.PeerCertificates) > 0 {
			opts := x509.VerifyOptions{
				Roots:     rhCACertPool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}
			if _, err := req.TLS.PeerCertificates[0].Verify(opts); err != nil {
				user = req.TLS.PeerCertificates[0].Subject.CommonName
				src = "Client-Cert-CN"
			} else {
				user = req.Header.Get("X-Remote-User")
				src = "X-Remote-user"
			}
		}
		vars := mux.Vars(req)
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "Resource %v requested by user[%s]=%s\n", vars["resource"], src, user)
	})
	srv.ListenAndServe(r)
}

func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
