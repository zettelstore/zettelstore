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
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/adapter/api"
	"zettelstore.de/z/web/adapter/webui"
	"zettelstore.de/z/web/server"
)

// ---------- Subcommand: run ------------------------------------------------

func flgRun(fs *flag.FlagSet) {
	fs.String("c", defConfigfile, "configuration file")
	fs.Uint("a", 0, "port number kernel service (0=disable)")
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
	kernel.Main.WaitForShutdown()
	return exitCode, err
}

func doRun(debug bool) (int, error) {
	kern := kernel.Main
	kern.SetDebug(debug)
	if err := kern.StartService(kernel.WebService); err != nil {
		return 1, err
	}
	return 0, nil
}

func setupRouting(webSrv server.Server, boxManager box.Manager, authManager auth.Manager, rtConfig config.Config) {
	protectedBoxManager, authPolicy := authManager.BoxWithPolicy(webSrv, boxManager, rtConfig)
	api := api.New(webSrv, authManager, authManager, webSrv, rtConfig)
	wui := webui.New(webSrv, authManager, rtConfig, authManager, boxManager, authPolicy)

	ucAuthenticate := usecase.NewAuthenticate(authManager, authManager, boxManager)
	ucCreateZettel := usecase.NewCreateZettel(rtConfig, protectedBoxManager)
	ucGetMeta := usecase.NewGetMeta(protectedBoxManager)
	ucGetAllMeta := usecase.NewGetAllMeta(protectedBoxManager)
	ucGetZettel := usecase.NewGetZettel(protectedBoxManager)
	ucParseZettel := usecase.NewParseZettel(rtConfig, ucGetZettel)
	ucListMeta := usecase.NewListMeta(protectedBoxManager)
	ucListRoles := usecase.NewListRole(protectedBoxManager)
	ucListTags := usecase.NewListTags(protectedBoxManager)
	ucZettelContext := usecase.NewZettelContext(protectedBoxManager)
	ucDelete := usecase.NewDeleteZettel(protectedBoxManager)
	ucUpdate := usecase.NewUpdateZettel(protectedBoxManager)

	webSrv.Handle("/", wui.MakeGetRootHandler(protectedBoxManager))
	webSrv.AddListRoute('a', http.MethodGet, wui.MakeGetLoginHandler())
	webSrv.AddListRoute('a', http.MethodPost, adapter.MakePostLoginHandler(
		api.MakePostLoginHandlerAPI(ucAuthenticate),
		wui.MakePostLoginHandlerHTML(ucAuthenticate)))
	webSrv.AddListRoute('a', http.MethodPut, api.MakeRenewAuthHandler())
	webSrv.AddZettelRoute('a', http.MethodGet, wui.MakeGetLogoutHandler())
	if !authManager.IsReadonly() {
		webSrv.AddZettelRoute('b', http.MethodGet, wui.MakeGetRenameZettelHandler(ucGetMeta))
		webSrv.AddZettelRoute('b', http.MethodPost, wui.MakePostRenameZettelHandler(
			usecase.NewRenameZettel(protectedBoxManager)))
		webSrv.AddZettelRoute('c', http.MethodGet, wui.MakeGetCopyZettelHandler(
			ucGetZettel, usecase.NewCopyZettel()))
		webSrv.AddZettelRoute('c', http.MethodPost, wui.MakePostCreateZettelHandler(ucCreateZettel))
		webSrv.AddZettelRoute('d', http.MethodGet, wui.MakeGetDeleteZettelHandler(ucGetZettel))
		webSrv.AddZettelRoute('d', http.MethodPost, wui.MakePostDeleteZettelHandler(ucDelete))
		webSrv.AddZettelRoute('e', http.MethodGet, wui.MakeEditGetZettelHandler(ucGetZettel))
		webSrv.AddZettelRoute('e', http.MethodPost, wui.MakeEditSetZettelHandler(ucUpdate))
		webSrv.AddZettelRoute('f', http.MethodGet, wui.MakeGetFolgeZettelHandler(
			ucGetZettel, usecase.NewFolgeZettel(rtConfig)))
		webSrv.AddZettelRoute('f', http.MethodPost, wui.MakePostCreateZettelHandler(ucCreateZettel))
		webSrv.AddZettelRoute('g', http.MethodGet, wui.MakeGetNewZettelHandler(
			ucGetZettel, usecase.NewNewZettel()))
		webSrv.AddZettelRoute('g', http.MethodPost, wui.MakePostCreateZettelHandler(ucCreateZettel))
	}
	webSrv.AddListRoute('f', http.MethodGet, wui.MakeSearchHandler(
		usecase.NewSearch(protectedBoxManager), ucGetMeta, ucGetZettel))
	webSrv.AddListRoute('h', http.MethodGet, wui.MakeListHTMLMetaHandler(
		ucListMeta, ucListRoles, ucListTags))
	webSrv.AddZettelRoute('h', http.MethodGet, wui.MakeGetHTMLZettelHandler(
		ucParseZettel, ucGetMeta))
	webSrv.AddZettelRoute('i', http.MethodGet, wui.MakeGetInfoHandler(
		ucParseZettel, ucGetMeta, ucGetAllMeta))
	webSrv.AddZettelRoute('j', http.MethodGet, wui.MakeZettelContextHandler(ucZettelContext))

	webSrv.AddZettelRoute('l', http.MethodGet, api.MakeGetLinksHandler(ucParseZettel))
	webSrv.AddZettelRoute('o', http.MethodGet, api.MakeGetOrderHandler(
		usecase.NewZettelOrder(protectedBoxManager, ucParseZettel)))
	webSrv.AddListRoute('r', http.MethodGet, api.MakeListRoleHandler(ucListRoles))
	webSrv.AddListRoute('t', http.MethodGet, api.MakeListTagsHandler(ucListTags))
	webSrv.AddZettelRoute('y', http.MethodGet, api.MakeZettelContextHandler(ucZettelContext))
	webSrv.AddListRoute('z', http.MethodGet, api.MakeListMetaHandler(
		usecase.NewListMeta(protectedBoxManager), ucGetMeta, ucParseZettel))
	webSrv.AddZettelRoute('z', http.MethodGet, api.MakeGetZettelHandler(
		ucParseZettel, ucGetMeta))
	if !authManager.IsReadonly() {
		webSrv.AddListRoute('z', http.MethodPost, api.MakePostCreateZettelHandler(ucCreateZettel))
		webSrv.AddZettelRoute('z', http.MethodDelete, api.MakeDeleteZettelHandler(ucDelete))
		webSrv.AddZettelRoute('z', http.MethodPut, api.MakeUpdateZettelHandler(ucUpdate))
	}

	if authManager.WithAuth() {
		webSrv.SetUserRetriever(usecase.NewGetUserByZid(boxManager))
	}
}
