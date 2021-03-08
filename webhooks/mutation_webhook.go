package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"k8s.io/apimachinery/pkg/util/json"

	"github.com/gorilla/mux"
	mycrd "github.com/pkbhowmick/k8s-crd/pkg/apis/stable.example.com/v1alpha1"
	"github.com/spf13/afero"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/certstore"
	"github.com/tamalsaha/DIY-k8s-extended-apiserver/lib/server"
	adv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/util/cert"
)

var (
	runtimeSchema = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeSchema)
	deserializer  = codecs.UniversalDeserializer()
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:",omitempty"`
}

func CreatePatch(kubeapi *mycrd.KubeApi) {
	var patch []patchOperation
	patch = append()
}

func main() {
	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs, "../crt")
	handleErr(err)

	err = store.InitCA("apiserver")
	handleErr(err)
	serverCert, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.3")},
	})
	handleErr(err)
	err = store.Write("webtls", serverCert, serverKey)
	handleErr(err)

	cfg := server.Config{
		Address: "127.0.0.3:8443",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile: store.CertFile("webtls"),
		KeyFile:  store.KeyFile("webtls"),
	}
	srv := server.NewGenericServer(cfg)

	r := mux.NewRouter()
	r.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(res, "Webhook server is ok")
	})
	r.HandleFunc("/mutate", Mutator)
	srv.ListenAndServe(r)
}

func Mutator(res http.ResponseWriter, req *http.Request) {
	var body []byte
	if req.Body != nil {
		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		http.Error(res, "empty body", http.StatusBadRequest)
		return
	}
	// check content-type later

	var admissionReq adv1.AdmissionReview
	var admissionRes *adv1.AdmissionResponse
	if _, _, err := deserializer.Decode(body, nil, &admissionReq); err != nil {
		admissionRes = &adv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionRes = Mutate(&admissionReq)
	}
}

func Mutate(admissionReq *adv1.AdmissionRequest) *adv1.AdmissionResponse {
	req := admissionReq.Object
	var kubeapi mycrd.KubeApi
	if err := json.Unmarshal(req.Raw, &kubeapi); err != nil {
		return &adv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}
	var res *adv1.AdmissionResponse
	if kubeapi.Spec.DeploymentName == "" {
		res.Patch
	}
	if kubeapi.Spec.ServiceName == "" {
		kubeapi.Spec.ServiceName = "SomeRandomName"
	}

}

func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
