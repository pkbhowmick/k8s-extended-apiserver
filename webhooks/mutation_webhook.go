package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	mycrd "github.com/pkbhowmick/k8s-crd/pkg/apis/stable.example.com/v1alpha1"
	adv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
)

var (
	runtimeSchema = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeSchema)
	deserializer  = codecs.UniversalDeserializer()
)

type ServerParameters struct {
	certFile string
	keyFile  string
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:",omitempty"`
}

func main() {
	var parameters ServerParameters
	flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(parameters.certFile, parameters.keyFile)
	if err != nil {
		log.Fatalln(err)
		return
	}

	r := mux.NewRouter()
	r.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(res, "Webhook server is ok")
	})
	r.HandleFunc("/mutate", Serve)
	var server *http.Server
	server = &http.Server{
		Addr:      ":443",
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		Handler:   r,
	}
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Fatalf("Failed to listen and serve webhook server: %v", err)
		}
	}()

}

func Serve(res http.ResponseWriter, req *http.Request) {
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

	ar := adv1.AdmissionReview{}
	var admissionRes *adv1.AdmissionResponse
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		admissionRes = &adv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionRes = Mutate(&ar)
	}
	admissionRev := adv1.AdmissionReview{}
	if admissionRes != nil {
		admissionRev.Response = admissionRes
		if ar.Request != nil {
			admissionRev.Response.UID = ar.Request.UID
		}
	}
	resp, err := json.Marshal(admissionRev)
	if err != nil {
		http.Error(res, "can't encode response", http.StatusInternalServerError)
	}
	res.Write(resp)
}

func Mutate(admissionRev *adv1.AdmissionReview) *adv1.AdmissionResponse {
	req := admissionRev.Request
	var kubeapi mycrd.KubeApi
	if err := json.Unmarshal(req.Object.Raw, &kubeapi); err != nil {
		return &adv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}
	var patch []patchOperation
	if kubeapi.Spec.DeploymentName == "" {
		patch = append(patch, patchOperation{
			Op:    "replace",
			Path:  "/spec/deploymentName",
			Value: "somerandomdeploymentname",
		})
	}
	if kubeapi.Spec.ServiceName == "" {
		patch = append(patch, patchOperation{
			Op:    "replace",
			Path:  "/spec/serviceName",
			Value: "somerandomservicename",
		})
	}
	patchBytes, err := json.Marshal(patch)
	handleErr(err)

	return &adv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *adv1.PatchType {
			pt := adv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func handleErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
