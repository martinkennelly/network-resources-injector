package server

import (
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/webhook"
	"net/http"
)

// httpServerHandler limits HTTP server endpoint to /mutate and HTTP verb to POST only
func httpServerHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != endpoint {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid HTTP verb requested", 405)
		return
	}
	webhook.MutateHandler(w, r)
}

