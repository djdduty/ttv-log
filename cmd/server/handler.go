package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/djdduty/ttv-log/config"
	"github.com/gorilla/context"
	"github.com/gorilla/csrf"
	"github.com/julienschmidt/httprouter"
	negronilogrus "github.com/meatballhat/negroni-logrus"
	"github.com/ory/graceful"
	"github.com/spf13/cobra"
	"github.com/unrolled/render"
	"github.com/unrolled/secure"
	"github.com/urfave/negroni"
)

// EnhanceRouter enhances router with configured middleware
func EnhanceRouter(c *config.Config, serverHandler *Handler, router *httprouter.Router, middlewares []negroni.Handler, enableCors, rejectInsecure bool) http.Handler {
	n := negroni.New()
	for _, m := range middlewares {
		n.Use(m)
	}

	if rejectInsecure {
		n.UseFunc(serverHandler.RejectInsecureRequests)
	}

	CSRF := csrf.Protect(
		[]byte("32-byte-long-auth-key"), // TODO: Read CSRF auth token from config
		csrf.FieldName("_csrf_token"),
		csrf.CookieName("_csrf_token"),
		csrf.Secure(!c.ForceHTTP),
	)

	n.UseHandler(CSRF(router))
	return context.ClearHandler(n)
}

// RunServe runs the serve command after setting up middleware and handlers
func RunServe(c *config.Config) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		serverHandler, router, middlewares := setup(c, cmd, args)

		address := c.GetAddress()
		serve(c, cmd, EnhanceRouter(c, serverHandler, router, middlewares, false, !addressIsUnixSocket(address)), address, nil)
	}
}

func setup(c *config.Config, cmd *cobra.Command, args []string) (handler *Handler, router *httprouter.Router, middlewares []negroni.Handler) {
	router = httprouter.New()

	w := render.New(render.Options{
		/*Funcs: []template.FuncMap{
			template.FuncMap{"noescape": noescape},
			//template.FuncMap{"getScopeDescription": consent.GetScopeDescription},
			template.FuncMap{"eq": func(a, b interface{}) bool { return a == b }},
		},
		Layout: "layout",*/
	})

	handler = NewHandler(c, w)
	handler.RegisterRoutes(router)
	var err error
	c.ForceHTTP, err = cmd.Flags().GetBool("dangerous-force-http")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// setup middlewares for tracing, telementry, etc
	secureMiddleware := secure.New(secure.Options{
		AllowedHosts:          []string{"ttvlog.djdduty.com"},
		SSLRedirect:           true,
		SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:            315360000,
		STSIncludeSubdomains:  true,
		STSPreload:            true,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'self' *; img-src *",
		IsDevelopment:         c.ForceHTTP,
	})

	recovery := negroni.NewRecovery()
	recovery.PrintStack = false

	static := negroni.NewStatic(http.Dir("client/build"))
	static.Prefix = "/static"

	middlewares = append(
		middlewares,
		negronilogrus.NewMiddlewareFromLogger(c.GetLogger(), "API"),
		recovery,
		negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext),
		static,
	)

	return
}

func noescape(str string) template.HTML {
	return template.HTML(str)
}

// Handler ...
type Handler struct {
	R      *render.Render
	Config *config.Config
}

// NewHandler creates a new handler instance
func NewHandler(c *config.Config, r *render.Render) *Handler {
	return &Handler{Config: c, R: r}
}

// RegisterRoutes registers all the handler's routes
func (h *Handler) RegisterRoutes(router *httprouter.Router) {
	c := h.Config
	// Setup handlers for all modules
	newHealthHandler(c, router, h.R)
	newAPIHandler(c, router, h.R)
}

// RejectInsecureRequests is a middleware for denying requests that don't fit the secure scheme
func (h *Handler) RejectInsecureRequests(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.TLS != nil || h.Config.ForceHTTP {
		next.ServeHTTP(rw, r)
		return
	}

	if err := h.Config.DoesRequestSatisfyTermination(r); err == nil {
		next.ServeHTTP(rw, r)
		//return
	} else {
		h.Config.GetLogger().WithError(err).Warnln("Could not serve http connection")
	}

	h.R.JSON(rw, http.StatusBadGateway, errors.New("Can not serve request over insecure http"))
}

func serve(c *config.Config, cmd *cobra.Command, handler http.Handler, address string, cert []tls.Certificate) {
	var srv = graceful.WithDefaults(&http.Server{
		Addr:    address,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: cert,
		},
	})

	err := graceful.Graceful(func() error {
		var err error
		c.GetLogger().Infof("Setting up http server on %s", address)
		if addressIsUnixSocket(address) {
			addr := strings.TrimPrefix(address, "unix:")
			unixListener, e := net.Listen("unix", addr)
			if e != nil {
				return e
			}
			err = srv.Serve(unixListener)
		} else {
			if c.ForceHTTP {
				c.GetLogger().Warnln("HTTPS disabled, never do this in production.")
				err = srv.ListenAndServe()
			} else if c.AllowTLSTermination != "" {
				c.GetLogger().Infoln("TLS termination enabled, disabling https.")
				err = srv.ListenAndServe()
			} else {
				err = srv.ListenAndServeTLS("", "")
			}
		}

		return err
	}, srv.Shutdown)
	if err != nil {
		c.GetLogger().WithError(err).Fatal("Could not gracefully run server")
	}
}

func addressIsUnixSocket(address string) bool {
	return strings.HasPrefix(address, "unix:")
}
