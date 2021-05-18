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
	"net/http"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/service"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/adapter/api"
	"zettelstore.de/z/web/adapter/webui"
	"zettelstore.de/z/web/server"
)

// ---------- Subcommand: run ------------------------------------------------

func flgRun(fs *flag.FlagSet) {
	fs.String("c", defConfigfile, "configuration file")
	fs.Uint("a", 0, "port number core service (0=disable)")
	fs.Uint("p", 23123, "port number web service")
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
	exitCode, err := doRun(withDebug(fs))
	service.Main.WaitForShutdown()
	return exitCode, err
}

func doRun(debug bool) (int, error) {
	srvm := service.Main
	srvm.SetDebug(debug)
	if err := srvm.StartSub(service.SubWeb); err != nil {
		return 1, err
	}
	return 0, nil
}

func setupRouting(webSrv server.Server, placeManager place.Manager, authManager auth.Manager) {
	protectedPlaceManager, authPolicy := authManager.PlaceWithPolicy(placeManager)
	te := webui.NewTemplateEngine(webSrv, authManager, placeManager, authPolicy)

	ucAuthenticate := usecase.NewAuthenticate(authManager, placeManager)
	ucGetMeta := usecase.NewGetMeta(protectedPlaceManager)
	ucGetZettel := usecase.NewGetZettel(protectedPlaceManager)
	ucParseZettel := usecase.NewParseZettel(ucGetZettel)
	ucListMeta := usecase.NewListMeta(protectedPlaceManager)
	ucListRoles := usecase.NewListRole(protectedPlaceManager)
	ucListTags := usecase.NewListTags(protectedPlaceManager)
	ucZettelContext := usecase.NewZettelContext(protectedPlaceManager)

	webSrv.Handle("/", webui.MakeGetRootHandler(authManager, te, protectedPlaceManager))
	webSrv.AddListRoute('a', http.MethodGet, webui.MakeGetLoginHandler(te))
	webSrv.AddListRoute('a', http.MethodPost, adapter.MakePostLoginHandler(
		api.MakePostLoginHandlerAPI(authManager, ucAuthenticate),
		webui.MakePostLoginHandlerHTML(authManager, te, ucAuthenticate)))
	webSrv.AddListRoute('a', http.MethodPut, api.MakeRenewAuthHandler())
	webSrv.AddZettelRoute('a', http.MethodGet, webui.MakeGetLogoutHandler(te))
	if !authManager.IsReadonly() {
		webSrv.AddZettelRoute('b', http.MethodGet, webui.MakeGetRenameZettelHandler(
			te, ucGetMeta))
		webSrv.AddZettelRoute('b', http.MethodPost, webui.MakePostRenameZettelHandler(
			te, usecase.NewRenameZettel(protectedPlaceManager)))
		webSrv.AddZettelRoute('c', http.MethodGet, webui.MakeGetCopyZettelHandler(
			te, ucGetZettel, usecase.NewCopyZettel()))
		webSrv.AddZettelRoute('c', http.MethodPost, webui.MakePostCreateZettelHandler(
			te, usecase.NewCreateZettel(protectedPlaceManager)))
		webSrv.AddZettelRoute('d', http.MethodGet, webui.MakeGetDeleteZettelHandler(
			te, ucGetZettel))
		webSrv.AddZettelRoute('d', http.MethodPost, webui.MakePostDeleteZettelHandler(
			te, usecase.NewDeleteZettel(protectedPlaceManager)))
		webSrv.AddZettelRoute('e', http.MethodGet, webui.MakeEditGetZettelHandler(
			te, ucGetZettel))
		webSrv.AddZettelRoute('e', http.MethodPost, webui.MakeEditSetZettelHandler(
			te, usecase.NewUpdateZettel(protectedPlaceManager)))
		webSrv.AddZettelRoute('f', http.MethodGet, webui.MakeGetFolgeZettelHandler(
			te, ucGetZettel, usecase.NewFolgeZettel()))
		webSrv.AddZettelRoute('f', http.MethodPost, webui.MakePostCreateZettelHandler(
			te, usecase.NewCreateZettel(protectedPlaceManager)))
		webSrv.AddZettelRoute('g', http.MethodGet, webui.MakeGetNewZettelHandler(
			te, ucGetZettel, usecase.NewNewZettel()))
		webSrv.AddZettelRoute('g', http.MethodPost, webui.MakePostCreateZettelHandler(
			te, usecase.NewCreateZettel(protectedPlaceManager)))
	}
	webSrv.AddListRoute('f', http.MethodGet, webui.MakeSearchHandler(
		te, usecase.NewSearch(protectedPlaceManager), ucGetMeta, ucGetZettel))
	webSrv.AddListRoute('h', http.MethodGet, webui.MakeListHTMLMetaHandler(
		te, ucListMeta, ucListRoles, ucListTags))
	webSrv.AddZettelRoute('h', http.MethodGet, webui.MakeGetHTMLZettelHandler(
		te, ucParseZettel, ucGetMeta))
	webSrv.AddZettelRoute('i', http.MethodGet, webui.MakeGetInfoHandler(
		te, ucParseZettel, ucGetMeta))
	webSrv.AddZettelRoute('j', http.MethodGet, webui.MakeZettelContextHandler(te, ucZettelContext))

	webSrv.AddZettelRoute('l', http.MethodGet, api.MakeGetLinksHandler(ucParseZettel))
	webSrv.AddZettelRoute('o', http.MethodGet, api.MakeGetOrderHandler(
		usecase.NewZettelOrder(protectedPlaceManager, ucParseZettel)))
	webSrv.AddListRoute('r', http.MethodGet, api.MakeListRoleHandler(ucListRoles))
	webSrv.AddListRoute('t', http.MethodGet, api.MakeListTagsHandler(ucListTags))
	webSrv.AddZettelRoute('y', http.MethodGet, api.MakeZettelContextHandler(ucZettelContext))
	webSrv.AddListRoute('z', http.MethodGet, api.MakeListMetaHandler(
		usecase.NewListMeta(protectedPlaceManager), ucGetMeta, ucParseZettel))
	webSrv.AddZettelRoute('z', http.MethodGet, api.MakeGetZettelHandler(ucParseZettel, ucGetMeta))

	webSrv.SetUserRetriever(usecase.NewGetUserByZid(placeManager))
}
