//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"flag"
	"log"
	"net/http"

	"zettelstore.de/z/auth/policy"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/service"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/adapter/api"
	"zettelstore.de/z/web/adapter/webui"
	"zettelstore.de/z/web/router"
	"zettelstore.de/z/web/session"
)

// ---------- Subcommand: run ------------------------------------------------

func flgRun(fs *flag.FlagSet) {
	fs.String("c", defConfigfile, "configuration file")
	fs.Uint("p", 23123, "port number")
	fs.String("d", "", "zettel directory")
	fs.Bool("r", false, "system-wide read-only mode")
	fs.Bool("v", false, "verbose mode")
	fs.Bool("debug", false, "debug mode")
}

func withDebug(fs *flag.FlagSet) bool {
	dbg := fs.Lookup("debug")
	return dbg != nil && dbg.Value.String() == "true"
}

func runFunc(fs *flag.FlagSet, cfg *meta.Meta) (int, error) {
	setupWebConfig(cfg)
	exitCode, err := doRun(withDebug(fs))
	service.Main.WaitForShutdown()
	return exitCode, err
}

func setupWebConfig(cfg *meta.Meta) {
	srvm := service.Main
	if val, ok := cfg.Get("listen-addr"); ok {
		srvm.SetConfig(service.SubWeb, service.WebListenAddress, val)
	}
	prefix, ok := cfg.Get("url-prefix")
	if ok && len(prefix) > 0 && prefix[0] == '/' && prefix[len(prefix)-1] == '/' {
		srvm.SetConfig(service.SubWeb, service.WebURLPrefix, prefix)
	}

}

func doRun(debug bool) (int, error) {
	srvm := service.Main
	readonlyMode := startup.IsReadOnlyMode()
	v := startup.GetVersion()
	log.Printf("%v %v (%v@%v/%v)", v.Prog, v.Build, v.GoVersion, v.Os, v.Arch)
	log.Println("Licensed under the latest version of the EUPL (European Union Public License)")
	log.Printf("Listening on %v", srvm.GetConfig(service.SubWeb, service.WebListenAddress))
	log.Printf("Zettel location [%v]", startup.PlaceManager().Location())
	if readonlyMode {
		log.Println("Read-only mode")
	}

	srvm.SetDebug(debug)
	srvm.WebSetConfig(func(urlPrefix string) http.Handler {
		return setupRouting(urlPrefix, startup.PlaceManager(), readonlyMode)
	})
	if err := srvm.WebStart(); err != nil {
		return 1, err
	}
	return 0, nil

}

func setupRouting(urlPrefix string, mgr place.Manager, readonlyMode bool) http.Handler {
	router := router.NewRouter(urlPrefix)
	var up place.Place = mgr
	pp, pol := policy.PlaceWithPolicy(
		up, startup.IsSimple(), startup.WithAuth, readonlyMode, runtime.GetExpertMode,
		startup.IsOwner, runtime.GetVisibility)
	te := webui.NewTemplateEngine(mgr, pol, router.NewURLBuilder)

	ucAuthenticate := usecase.NewAuthenticate(up)
	ucGetMeta := usecase.NewGetMeta(pp)
	ucGetZettel := usecase.NewGetZettel(pp)
	ucParseZettel := usecase.NewParseZettel(ucGetZettel)
	ucListMeta := usecase.NewListMeta(pp)
	ucListRoles := usecase.NewListRole(pp)
	ucListTags := usecase.NewListTags(pp)
	ucZettelContext := usecase.NewZettelContext(pp)

	router.Handle("/", webui.MakeGetRootHandler(te, pp))
	router.AddListRoute('a', http.MethodGet, webui.MakeGetLoginHandler(te))
	router.AddListRoute('a', http.MethodPost, adapter.MakePostLoginHandler(
		api.MakePostLoginHandlerAPI(ucAuthenticate),
		webui.MakePostLoginHandlerHTML(te, ucAuthenticate)))
	router.AddListRoute('a', http.MethodPut, api.MakeRenewAuthHandler())
	router.AddZettelRoute('a', http.MethodGet, webui.MakeGetLogoutHandler(te))
	if !readonlyMode {
		router.AddZettelRoute('b', http.MethodGet, webui.MakeGetRenameZettelHandler(
			te, ucGetMeta))
		router.AddZettelRoute('b', http.MethodPost, webui.MakePostRenameZettelHandler(
			te, usecase.NewRenameZettel(pp)))
		router.AddZettelRoute('c', http.MethodGet, webui.MakeGetCopyZettelHandler(
			te, ucGetZettel, usecase.NewCopyZettel()))
		router.AddZettelRoute('c', http.MethodPost, webui.MakePostCreateZettelHandler(
			te, usecase.NewCreateZettel(pp)))
		router.AddZettelRoute('d', http.MethodGet, webui.MakeGetDeleteZettelHandler(
			te, ucGetZettel))
		router.AddZettelRoute('d', http.MethodPost, webui.MakePostDeleteZettelHandler(
			te, usecase.NewDeleteZettel(pp)))
		router.AddZettelRoute('e', http.MethodGet, webui.MakeEditGetZettelHandler(
			te, ucGetZettel))
		router.AddZettelRoute('e', http.MethodPost, webui.MakeEditSetZettelHandler(
			te, usecase.NewUpdateZettel(pp)))
		router.AddZettelRoute('f', http.MethodGet, webui.MakeGetFolgeZettelHandler(
			te, ucGetZettel, usecase.NewFolgeZettel()))
		router.AddZettelRoute('f', http.MethodPost, webui.MakePostCreateZettelHandler(
			te, usecase.NewCreateZettel(pp)))
		router.AddZettelRoute('g', http.MethodGet, webui.MakeGetNewZettelHandler(
			te, ucGetZettel, usecase.NewNewZettel()))
		router.AddZettelRoute('g', http.MethodPost, webui.MakePostCreateZettelHandler(
			te, usecase.NewCreateZettel(pp)))
	}
	router.AddListRoute('f', http.MethodGet, webui.MakeSearchHandler(
		te, usecase.NewSearch(pp), ucGetMeta, ucGetZettel))
	router.AddListRoute('h', http.MethodGet, webui.MakeListHTMLMetaHandler(
		te, ucListMeta, ucListRoles, ucListTags))
	router.AddZettelRoute('h', http.MethodGet, webui.MakeGetHTMLZettelHandler(
		te, ucParseZettel, ucGetMeta))
	router.AddZettelRoute('i', http.MethodGet, webui.MakeGetInfoHandler(
		te, ucParseZettel, ucGetMeta))
	router.AddZettelRoute('j', http.MethodGet, webui.MakeZettelContextHandler(te, ucZettelContext))

	router.AddZettelRoute('l', http.MethodGet, api.MakeGetLinksHandler(ucParseZettel))
	router.AddZettelRoute('o', http.MethodGet, api.MakeGetOrderHandler(
		usecase.NewZettelOrder(pp, ucParseZettel)))
	router.AddListRoute('r', http.MethodGet, api.MakeListRoleHandler(ucListRoles))
	router.AddListRoute('t', http.MethodGet, api.MakeListTagsHandler(ucListTags))
	router.AddZettelRoute('y', http.MethodGet, api.MakeZettelContextHandler(ucZettelContext))
	router.AddListRoute('z', http.MethodGet, api.MakeListMetaHandler(
		usecase.NewListMeta(pp), ucGetMeta, ucParseZettel))
	router.AddZettelRoute('z', http.MethodGet, api.MakeGetZettelHandler(ucParseZettel, ucGetMeta))
	return session.NewHandler(router, usecase.NewGetUserByZid(up))
}
