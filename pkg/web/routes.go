package web

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/feedhenry/mcp-standalone/pkg/mobile"
	"github.com/feedhenry/mcp-standalone/pkg/web/middleware"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kerror "k8s.io/apimachinery/pkg/api/errors"
)

// NewRouter sets up the HTTP Router
func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	return r
}

// BuildHTTPHandler puts together our HTTPHandler
func BuildHTTPHandler(r *mux.Router, access *middleware.Access) http.Handler {
	recovery := negroni.NewRecovery()
	recovery.PrintStack = false
	n := negroni.New(recovery)
	cors := middleware.Cors{}
	n.UseFunc(cors.Handle)
	if access != nil {
		n.UseFunc(access.Handle)
	} else {
		fmt.Println("access control is turned off ")
	}
	r.Handle("/metrics", promhttp.Handler())
	n.UseHandler(r)
	return n
}

func ConsoleConfigRoute(r *mux.Router, handler *ConsoleConfigHandler) {
	r.HandleFunc("/config.js", handler.Config)
}

// StaticRoute configures & sets up the /console route.
func StaticRoute(handler *StaticHandler) {
	http.HandleFunc(handler.prefix+"/", handler.Static)
}

// OAuthRoute configures & sets up the /oauth route.
func OAuthRoute(r *mux.Router, handler *OAuthHandler) {
	r.HandleFunc("/oauth/token", handler.OAuthToken)
}

// MobileAppRoute configure and setup the /mobileapp route. The middleware.Builder is responsible for building per request instances of clients
func MobileAppRoute(r *mux.Router, handler *MobileAppHandler) {
	r.HandleFunc("/mobileapp", prometheus.InstrumentHandlerFunc("mobileapp create", handler.Create)).Methods("POST")
	r.HandleFunc("/mobileapp/{id}", prometheus.InstrumentHandlerFunc("mobileapp delete", handler.Delete)).Methods("DELETE")
	r.HandleFunc("/mobileapp/{id}", prometheus.InstrumentHandlerFunc("mobileapp read", handler.Read)).Methods("GET")
	r.HandleFunc("/mobileapp", prometheus.InstrumentHandlerFunc("mobileapp list", handler.List)).Methods("GET")
	r.HandleFunc("/mobileapp/{id}", prometheus.InstrumentHandlerFunc("mobileapp update", handler.Update)).Methods("PUT")
}

//SDKConfigRoute configures and sets up the /sdk routes
func SDKConfigRoute(r *mux.Router, handler *SDKConfigHandler) {
	r.HandleFunc("/sdk/mobileapp/{id}/config", prometheus.InstrumentHandlerFunc("sdk config", handler.Read)).Methods("GET")
}

// SysRoute congifures and sets up the /sys/* route
func SysRoute(r *mux.Router, handler *SysHandler) {
	r.HandleFunc("/sys/info/ping", handler.Ping).Methods("GET")
	r.HandleFunc("/sys/info/health", handler.Health).Methods("GET")
}

// MobileServiceRoute configures and sets up the /mobileservice routes
func MobileServiceRoute(r *mux.Router, handler *MobileServiceHandler) {
	r.HandleFunc("/mobileservice", prometheus.InstrumentHandlerFunc("mobileservice create", handler.Create)).Methods("POST")
	r.HandleFunc("/mobileservice", prometheus.InstrumentHandlerFunc("mobileservices list", handler.List)).Methods("GET")
	r.HandleFunc("/mobileservice/{name}", prometheus.InstrumentHandlerFunc("mobileservice read", handler.Read)).Methods("GET")
	r.HandleFunc("/mobileservice/{name}", prometheus.InstrumentHandlerFunc("mobileservice delete ", handler.Delete)).Methods("DELETE")
	r.HandleFunc("/mobileservice/configure/{component}/{secret}", prometheus.InstrumentHandlerFunc("mobileservices configuration", handler.Configure)).Methods("POST")
	r.HandleFunc("/mobileservice/configure/{component}/{secret}", prometheus.InstrumentHandlerFunc("mobileservices remove configuration", handler.Deconfigure)).Methods("DELETE")
}

//TODO maybe better place to put this
func handleCommonErrorCases(err error, rw http.ResponseWriter, logger *logrus.Logger) {
	e := errors.Cause(err)
	if mobile.IsNotFoundError(err) {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	if e, ok := e.(*mobile.StatusError); ok {
		logger.Error(fmt.Sprintf("status error occurred %v", err))
		http.Error(rw, e.Error(), e.StatusCode())
		return
	}
	if e, ok := e.(*kerror.StatusError); ok {
		logger.Error(fmt.Sprintf("kubernetes status error occurred %v", err))
		http.Error(rw, e.Error(), int(e.Status().Code))
		return
	}
	logger.Error(fmt.Sprintf("unexpected and unknown error occurred %+v", err))
	http.Error(rw, err.Error(), http.StatusInternalServerError)
}
