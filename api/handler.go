package api

import (
	"net/http"

	"github.com/djdduty/ttv-log/config"
	"github.com/julienschmidt/httprouter"
	"github.com/unrolled/render"
)

const (
	// StreamPath ...
	StreamPath = "/streams"
	// UserPath ...
	UserPath = "/users"
)

// Handler handles the http requests to api endpoints
type Handler struct {
	R *render.Render
	E *config.ElasticConnector
}

// NewHandler instantiates a handler.
func NewHandler(r *render.Render, e *config.ElasticConnector) *Handler {
	return &Handler{
		R: r,
		E: e,
	}
}

// SetRoutes registers this handler's routes.
func (h *Handler) SetRoutes(r *httprouter.Router) {
	r.GET(StreamPath, h.ListStreams)
	r.GET(UserPath, h.ListUsers)
}

type healthStatus struct {
	Status string `json:"status"`
}

// ListStreams ...
func (h *Handler) ListStreams(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client := h.E.GetClient()
	ctx := h.E.GetContext()

	f := NewFinder()
	f = f.From(0).Size(100)
	f = f.Sort("Channel.keyword", "Message.keyword")
	f = f.Pretty(false)

	// Provide a timeout of 5 seconds
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()

	// Execute the finder
	res, err := f.Find(ctx, client)
	if err != nil {
		panic(err)
	}

	h.R.JSON(rw, http.StatusOK, &res.Channels)
}

// ListUsers ...
func (h *Handler) ListUsers(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	h.R.JSON(rw, http.StatusOK, &healthStatus{
		Status: "ok",
	})
}
