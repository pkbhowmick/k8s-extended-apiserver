package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

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

	err = store.InitCA("apiserver")
	handleErr(err)

	serverCert, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.1")},
	})
	handleErr(err)

	err = store.Write("tls", serverCert, serverKey)
	handleErr(err)

	clientCert, clientKey, err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"alice"},
	})
	handleErr(err)

	err = store.Write("alice", clientCert, clientKey)
	handleErr(err)

	// **************** Request Header Store *****************
	rhStore, err := certstore.NewCertStore(fs, "../crt")
	handleErr(err)

	err = rhStore.InitCA("requestheader")
	handleErr(err)

	rhClientCert, rhClientKey, err := rhStore.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"apiserver"},
	})
	handleErr(err)

	err = rhStore.Write("apiserver", rhClientCert, rhClientKey)
	handleErr(err)

	rhCert, err := tls.LoadX509KeyPair(rhStore.CertFile("apiserver"), rhStore.KeyFile("apiserver"))
	handleErr(err)

	// **************************************

	easCACertPool := x509.NewCertPool()
	easStore, err := certstore.NewCertStore(fs, "../crt")
	handleErr(err)

	err = easStore.LoadCA("database")
	handleErr(err)

	easCACertPool.AppendCertsFromPEM(easStore.CACertBytes())

	// *************************************

	cfg := server.Config{
		Address: "127.0.0.1:8080",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile: store.CertFile("tls"),
		KeyFile:  store.KeyFile("tls"),
	}
	cfg.CACertFiles = append(cfg.CACertFiles, easStore.CertFile("ca"))
	//fmt.Println(cfg.CACertFiles)
	srv := server.NewGenericServer(cfg)

	r := mux.NewRouter()
	r.HandleFunc("/core/{resource}", func(res http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "Resource: %v\n", vars["resource"])
	})
	r.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "All is good!")
	})
	r.HandleFunc("/database/{resource}", func(res http.ResponseWriter, req *http.Request) {
		tr := &http.Transport{
			MaxConnsPerHost: 10,
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{rhCert},
				RootCAs:      easCACertPool,
			},
		}
		client := http.Client{
			Transport: tr,
			Timeout:   time.Duration(30 * time.Second),
		}
		u := *req.URL
		u.Scheme = "https"
		u.Host = "127.0.0.2:8080"
		fmt.Printf("forwording request to %v\n", u.String())

		newReq, _ := http.NewRequest(req.Method, u.String(), nil)
		if len(req.TLS.PeerCertificates) == 0 {
			newErr := errors.New("user tls certificate missing")
			handleErr(newErr)
		}
		newReq.Header.Set("X-Remote-User", req.TLS.PeerCertificates[0].Subject.CommonName)
		resp, err := client.Do(newReq)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(res, "Error: %v\n", err.Error())
			return
		}
		defer resp.Body.Close()

		res.WriteHeader(http.StatusOK)
		io.Copy(res, resp.Body)
	})
	srv.ListenAndServe(r)
}

func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
