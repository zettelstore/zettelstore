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

	"zettelstore.de/z/auth"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/usecase"
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

func runFunc(fs *flag.FlagSet, _ *meta.Meta) (int, error) {
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
	a := api.New(webSrv, authManager, authManager, webSrv, rtConfig)
	wui := webui.New(webSrv, authManager, rtConfig, authManager, boxManager, authPolicy)

	ucAuthenticate := usecase.NewAuthenticate(authManager, authManager, boxManager)
	ucCreateZettel := usecase.NewCreateZettel(rtConfig, protectedBoxManager)
	ucGetMeta := usecase.NewGetMeta(protectedBoxManager)
	ucGetAllMeta := usecase.NewGetAllMeta(protectedBoxManager)
	ucGetZettel := usecase.NewGetZettel(protectedBoxManager)
	ucParseZettel := usecase.NewParseZettel(rtConfig, ucGetZettel)
	ucEvaluateZettel := usecase.NewEvaluateZettel(rtConfig, ucGetZettel, ucGetMeta)
	ucListMeta := usecase.NewListMeta(protectedBoxManager)
	ucListRoles := usecase.NewListRole(protectedBoxManager)
	ucListTags := usecase.NewListTags(protectedBoxManager)
	ucZettelContext := usecase.NewZettelContext(protectedBoxManager)
	ucDelete := usecase.NewDeleteZettel(protectedBoxManager)
	ucUpdate := usecase.NewUpdateZettel(protectedBoxManager)
	ucRename := usecase.NewRenameZettel(protectedBoxManager)

	webSrv.Handle("/", wui.MakeGetRootHandler(protectedBoxManager))

	// Web user interface
	if !authManager.IsReadonly() {
		webSrv.AddZettelRoute('b', server.MethodGet, wui.MakeGetRenameZettelHandler(ucGetMeta))
		webSrv.AddZettelRoute('b', server.MethodPost, wui.MakePostRenameZettelHandler(ucRename))
		webSrv.AddZettelRoute('c', server.MethodGet, wui.MakeGetCopyZettelHandler(
			ucGetZettel, usecase.NewCopyZettel()))
		webSrv.AddZettelRoute('c', server.MethodPost, wui.MakePostCreateZettelHandler(ucCreateZettel))
		webSrv.AddZettelRoute('d', server.MethodGet, wui.MakeGetDeleteZettelHandler(ucGetZettel))
		webSrv.AddZettelRoute('d', server.MethodPost, wui.MakePostDeleteZettelHandler(ucDelete))
		webSrv.AddZettelRoute('e', server.MethodGet, wui.MakeEditGetZettelHandler(ucGetZettel))
		webSrv.AddZettelRoute('e', server.MethodPost, wui.MakeEditSetZettelHandler(ucUpdate))
		webSrv.AddZettelRoute('f', server.MethodGet, wui.MakeGetFolgeZettelHandler(
			ucGetZettel, usecase.NewFolgeZettel(rtConfig)))
		webSrv.AddZettelRoute('f', server.MethodPost, wui.MakePostCreateZettelHandler(ucCreateZettel))
		webSrv.AddZettelRoute('g', server.MethodGet, wui.MakeGetNewZettelHandler(
			ucGetZettel, usecase.NewNewZettel()))
		webSrv.AddZettelRoute('g', server.MethodPost, wui.MakePostCreateZettelHandler(ucCreateZettel))
	}
	webSrv.AddListRoute('f', server.MethodGet, wui.MakeSearchHandler(
		usecase.NewSearch(protectedBoxManager), ucGetMeta, ucGetZettel))
	webSrv.AddListRoute('h', server.MethodGet, wui.MakeListHTMLMetaHandler(
		ucListMeta, ucListRoles, ucListTags))
	webSrv.AddZettelRoute('h', server.MethodGet, wui.MakeGetHTMLZettelHandler(
		ucEvaluateZettel, ucGetMeta))
	webSrv.AddListRoute('i', server.MethodGet, wui.MakeGetLoginHandler())
	webSrv.AddListRoute('i', server.MethodPost, wui.MakePostLoginHandler(ucAuthenticate))
	webSrv.AddZettelRoute('i', server.MethodGet, wui.MakeGetInfoHandler(
		ucParseZettel, ucGetMeta, ucGetAllMeta))
	webSrv.AddListRoute('j', server.MethodGet, wui.MakeGetLogoutHandler())
	webSrv.AddZettelRoute('j', server.MethodGet, wui.MakeZettelContextHandler(ucZettelContext))

	// API
	webSrv.AddListRoute('a', server.MethodPost, a.MakePostLoginHandler(ucAuthenticate))
	webSrv.AddListRoute('a', server.MethodPut, a.MakeRenewAuthHandler())
	webSrv.AddZettelRoute('l', server.MethodGet, a.MakeGetLinksHandler(ucEvaluateZettel))
	webSrv.AddZettelRoute('o', server.MethodGet, a.MakeGetOrderHandler(
		usecase.NewZettelOrder(protectedBoxManager, ucEvaluateZettel)))
	webSrv.AddListRoute('p', server.MethodGet, a.MakeListParsedMetaHandler('p', ucListMeta))
	webSrv.AddZettelRoute('p', server.MethodGet, a.MakeGetParsedZettelHandler(ucParseZettel))
	webSrv.AddListRoute('r', server.MethodGet, a.MakeListRoleHandler(ucListRoles))
	webSrv.AddListRoute('t', server.MethodGet, a.MakeListTagsHandler(ucListTags))
	webSrv.AddListRoute('v', server.MethodGet, a.MakeListParsedMetaHandler('v', ucListMeta))
	webSrv.AddZettelRoute('v', server.MethodGet, a.MakeGetEvalZettelHandler(ucEvaluateZettel))
	webSrv.AddZettelRoute('w', server.MethodGet, a.MakeGetRawZettelHandler(ucGetZettel))
	webSrv.AddZettelRoute('x', server.MethodGet, api.MakeZettelContextHandler(ucZettelContext))
	webSrv.AddListRoute('z', server.MethodGet, api.MakeListMetaHandler(ucListMeta))
	webSrv.AddZettelRoute('z', server.MethodGet, api.MakeGetZettelHandler(ucGetZettel))
	if !authManager.IsReadonly() {
		webSrv.AddListRoute('z', server.MethodPost, a.MakePostCreateZettelHandler(ucCreateZettel))
		webSrv.AddZettelRoute('z', server.MethodDelete, a.MakeDeleteZettelHandler(ucDelete))
		webSrv.AddZettelRoute('z', server.MethodPut, a.MakeUpdateZettelHandler(ucUpdate))
		webSrv.AddZettelRoute('z', server.MethodMove, a.MakeRenameZettelHandler(ucRename))
	}

	if authManager.WithAuth() {
		webSrv.SetUserRetriever(usecase.NewGetUserByZid(boxManager))
	}
}
