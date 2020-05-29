package main

import (
	"crypto/tls"
	"net/http"
	"strconv"

	"github.com/rmanna/wash/env"
	"github.com/rmanna/wash/globals"
	"github.com/rmanna/wash/routes"
	"github.com/rmanna/wash/translations"

	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
	"github.com/go-playground/pure"
	"github.com/go-playground/statics/static"
)

const (
	appPath = "$GOPATH/src/github.com/rmanna/wash"
)

func init() {
	cLog := console.New()
	log.RegisterHandler(cLog, log.AllLevels...)
}

func main() {

	validate := initValidator()

	cfg, err := env.Parse(validate)
	if err != nil {
		log.WithFields(log.F("error", err)).Alert("Error Parsing ENV variables")
	}

	assets, err := newStaticAssets(&static.Config{UseStaticFiles: cfg.IsProduction, FallbackToDisk: true, AbsPkgPath: appPath})
	if err != nil {
		log.WithFields(log.F("error", err)).Fatal("Issue initializing static assets")
	}

	ut := translations.Initialize(validate)

	templates, err := initTemplates(cfg, assets)
	if err != nil {
		log.StackTrace().Panic(err)
	}

	err = startLiveReloadServer(templates, cfg, assets)
	if err != nil {
		log.WithFields(log.F("error", err)).Error("could not start livereload server")
	}

	app := globals.NewApp(cfg, ut, validate, templates)

	p := pure.New()

	redir := routes.Initialize(p, app, cfg)

	log.Infof("Listening on port %d\n", cfg.AppPort)
	if cfg.IsProduction {

		go func() {
			err := http.ListenAndServe(":"+strconv.Itoa(cfg.RedirectPort), redir.Serve())
			if err != nil {
				log.WithFields(log.F("error", err)).Error("shutting down redirect http listener")
			}
		}()

		certs, err := newStaticCerts(&static.Config{UseStaticFiles: cfg.IsProduction, FallbackToDisk: true, AbsPkgPath: appPath})
		if err != nil {
			log.WithFields(log.F("error", err)).Fatal("Issue initializing static certs")
		}

		httpKey, err := certs.ReadFile("tls.key")
		if err != nil {
			log.WithFields(log.F("error", err)).Fatal("Issue loading tls key")
		}

		httpPem, err := certs.ReadFile("tls.pem")
		if err != nil {
			log.WithFields(log.F("error", err)).Fatal("Issue loading tls pem")
		}

		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{

				// NOTE: NOT ALL Cipher Suites that ban be used are avaialble as of Go 1.6.2

				// Mozilla Recommended - https://wiki.mozilla.org/Security/Server_Side_TLS#Recommended_configurations
				// list TLS 1.2 ciphers - http://security.stackexchange.com/questions/76993/now-that-it-is-2015-what-ssl-tls-cipher-suites-should-be-used-in-a-high-securit
				// TLS 1.2 browser support http://caniuse.com/#search=tls1.2
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			},
			PreferServerCipherSuites: true,
			Certificates:             make([]tls.Certificate, 1),
		}

		tlsConfig.Certificates[0], err = tls.X509KeyPair(httpPem, httpKey)
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}

		server := &http.Server{Addr: ":" + strconv.Itoa(cfg.AppPort), Handler: p.Serve(), TLSConfig: tlsConfig}
		err = server.ListenAndServeTLS("", "")
		if err != nil {
			log.WithFields(log.F("error", err)).Error("shutting down server")
		}

	} else {

		err := http.ListenAndServe(":"+strconv.Itoa(cfg.AppPort), p.Serve())
		if err != nil {
			log.WithFields(log.F("error", err)).Error("shutting down server")
		}
	}
}
