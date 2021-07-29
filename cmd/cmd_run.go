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
		ucEvaluateZettel, ucGetMeta, ucGetAllMeta))
	webSrv.AddListRoute('j', server.MethodGet, wui.MakeGetLogoutHandler())
	webSrv.AddZettelRoute('j', server.MethodGet, wui.MakeZettelContextHandler(ucZettelContext))

	// API
	webSrv.AddListRoute('a', server.MethodPost, api.MakePostLoginHandler(ucAuthenticate))
	webSrv.AddListRoute('a', server.MethodPut, api.MakeRenewAuthHandler())
	webSrv.AddZettelRoute('l', server.MethodGet, api.MakeGetLinksHandler(ucEvaluateZettel))
	webSrv.AddZettelRoute('o', server.MethodGet, api.MakeGetOrderHandler(
		usecase.NewZettelOrder(protectedBoxManager, ucEvaluateZettel)))
	webSrv.AddListRoute('r', server.MethodGet, api.MakeListRoleHandler(ucListRoles))
	webSrv.AddListRoute('t', server.MethodGet, api.MakeListTagsHandler(ucListTags))
	webSrv.AddZettelRoute('v', server.MethodGet, api.MakeGetEvalZettelHandler(ucEvaluateZettel))
	webSrv.AddZettelRoute('x', server.MethodGet, api.MakeZettelContextHandler(ucZettelContext))
	webSrv.AddListRoute('z', server.MethodGet, api.MakeListMetaHandler(
		usecase.NewListMeta(protectedBoxManager)))
	webSrv.AddZettelRoute('z', server.MethodGet, api.MakeGetZettelHandler(ucGetZettel))
	if !authManager.IsReadonly() {
		webSrv.AddListRoute('z', server.MethodPost, api.MakePostCreateZettelHandler(ucCreateZettel))
		webSrv.AddZettelRoute('z', server.MethodDelete, api.MakeDeleteZettelHandler(ucDelete))
		webSrv.AddZettelRoute('z', server.MethodPut, api.MakeUpdateZettelHandler(ucUpdate))
		webSrv.AddZettelRoute('z', server.MethodMove, api.MakeRenameZettelHandler(ucRename))
	}

	if authManager.WithAuth() {
		webSrv.SetUserRetriever(usecase.NewGetUserByZid(boxManager))
	}
}
