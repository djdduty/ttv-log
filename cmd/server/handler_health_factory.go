package server

import (
	"github.com/djdduty/ttv-log/config"
	"github.com/djdduty/ttv-log/health"
	"github.com/julienschmidt/httprouter"
	"github.com/unrolled/render"
)

func newHealthHandler(c *config.Config, router *httprouter.Router, w *render.Render) *health.Handler {
	ctx := c.Context()
	health.ExpectDependency(c.GetLogger(), ctx.ElasticConnection)

	h := health.NewHandler(w, c.BuildVersion, health.ReadyCheckers{
		"database": ctx.ElasticConnection.Ping,
	})

	h.SetRoutes(router)
	return h
}
