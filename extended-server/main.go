package main

import (
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
	if err != nil {
		log.Fatalln(err)
	}
	err = store.NewCA("database")
	if err != nil {
		log.Fatalln(err)
	}

	serverCert, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.2")},
	})
	if err != nil {
		log.Fatalln(err)
	}
	err = store.Write("tls", serverCert, serverKey)
	if err != nil {
		log.Fatalln(err)
	}
	clientCert, clientkey, err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"bob"},
	})
	if err != nil {
		log.Fatalln(err)
	}
	err = store.Write("bob", clientCert, clientkey)
	if err != nil {
		log.Fatalln(err)
	}

	cfg := server.Config{
		Address: "127.0.0.2:8443",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile: store.CertFile("tls"),
		KeyFile:  store.KeyFile("tls"),
	}
	srv := server.NewGenericServer(cfg)

	r := mux.NewRouter()
	r.HandleFunc("/database/{resource}", func(res http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "Resource: %v\n", vars["resource"])
	})
	r.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "Extended Server is OK")
	})
	srv.ListenAndServe(r)
}
