package server

import (
	"github.com/djdduty/ttv-log/api"
	"github.com/djdduty/ttv-log/config"
	"github.com/djdduty/ttv-log/health"
	"github.com/julienschmidt/httprouter"
	"github.com/unrolled/render"
)

func newAPIHandler(c *config.Config, router *httprouter.Router, w *render.Render) *api.Handler {
	ctx := c.Context()
	health.ExpectDependency(c.GetLogger(), ctx.ElasticConnection)

	h := api.NewHandler(w, ctx.ElasticConnection)
	h.SetRoutes(router)
	return h
}
