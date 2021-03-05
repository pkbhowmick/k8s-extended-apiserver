package main

import (
	"log"

	"github.com/spf13/afero"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/certstore"
)

func main() {
	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs, "./crt")
	if err != nil {
		log.Fatalln(err)
	}
	err = store.NewCA("apiserver")
	if err != nil {
		log.Fatalln(err)
	}
	//serverCert, serverkey, err := store.NewServerCertPair(cert.AltNames{
	//	IPs: []net.IP{net.ParseIP("127.0.0.1")},
	//})
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//err = store.Write("tls", serverCert, serverkey)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//clientCert, clientKey, err := store.NewClientCertPair(cert.AltName{
	//	DNSNames: []string{"john"},
	//})
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//err = store.write("john", clientCert, clientKey)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//cfg := server.Config{}
}
