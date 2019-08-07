package health

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/unrolled/render"
)

type healthStatus struct {
	Status string `json:"status"`
}

type notReadyStatus struct {
	Errors map[string]string `json:"errors"`
}

type version struct {
	Version string `json:"version"`
}

const (
	// AliveCheckPath is the path where information about the life state of the instance is provided.
	AliveCheckPath = "/health/alive"
	// ReadyCheckPath is the path where information about the ready state of the instance is provided.
	ReadyCheckPath = "/health/ready"
	// VersionPath is the path where information about the software version of the instance is provided.
	VersionPath = "/version"
)

// RoutesToObserve returns a string of all the available routes of this module.
func RoutesToObserve() []string {
	return []string{
		AliveCheckPath,
		ReadyCheckPath,
		VersionPath,
	}
}

// ReadyChecker should return an error if the component is not ready.
type ReadyChecker func() error

// ReadyCheckers is a map of the ReadyCheckers.
type ReadyCheckers map[string]ReadyChecker

// NoopReadyChecker is a noop, returns nil
func NoopReadyChecker() error {
	return nil
}

// Handler handles the http requests to health and version endpoints
type Handler struct {
	R             *render.Render
	VersionString string
	ReadyChecks   ReadyCheckers
}

// NewHandler instantiates a handler.
func NewHandler(r *render.Render, version string, readyChecks ReadyCheckers) *Handler {
	return &Handler{
		R:             r,
		VersionString: version,
		ReadyChecks:   readyChecks,
	}
}

// SetRoutes registers this handler's routes.
func (h *Handler) SetRoutes(r *httprouter.Router) {
	r.GET(AliveCheckPath, h.Alive)
	r.GET(ReadyCheckPath, h.Ready)
	r.GET(VersionPath, h.Version)
}

// Alive returns an ok status if the instance is ready to handle HTTP requests.
//
// swagger:route GET /health/alive health isInstanceAlive
//
// Check alive status
func (h *Handler) Alive(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	h.R.JSON(rw, http.StatusOK, &healthStatus{
		Status: "ok",
	})
}

// Ready returns an ok status if the instance is ready to handle HTTP requests and all ReadyCheckers are ok
//
// swagger:route GET /health/ready health isInstanceReady
//
// Check readiness status
func (h *Handler) Ready(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var notReady = notReadyStatus{
		Errors: map[string]string{},
	}

	for n, c := range h.ReadyChecks {
		if err := c(); err != nil {
			notReady.Errors[n] = err.Error()
		}
	}

	if len(notReady.Errors) > 0 {
		h.R.JSON(rw, http.StatusServiceUnavailable, notReady)
		return
	}

	h.R.JSON(rw, http.StatusOK, &healthStatus{
		Status: "ok",
	})
}

// Version returns this service's version.
//
// swagger:route GET /version version getVersion
//
// Get service version
func (h *Handler) Version(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	h.R.JSON(rw, http.StatusOK, &version{
		Version: h.VersionString,
	})
}
