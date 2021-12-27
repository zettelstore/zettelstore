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

func runFunc(*flag.FlagSet) (int, error) {
	var exitCode int
	err := kernel.Main.StartService(kernel.WebService)
	if err != nil {
		exitCode = 1
	}
	kernel.Main.WaitForShutdown()
	return exitCode, err
}

func setupRouting(webSrv server.Server, boxManager box.Manager, authManager auth.Manager, rtConfig config.Config) {
	protectedBoxManager, authPolicy := authManager.BoxWithPolicy(webSrv, boxManager, rtConfig)
	kern := kernel.Main
	webLog := kern.GetLogger(kernel.WebService)
	a := api.New(
		webLog.Clone().Str("adapter", "api").Child(),
		webSrv, authManager, authManager, webSrv, rtConfig)
	wui := webui.New(
		webLog.Clone().Str("adapter", "wui").Child(),
		webSrv, authManager, rtConfig, authManager, boxManager, authPolicy)

	authLog := kern.GetLogger(kernel.AuthService)
	ucLog := kern.GetLogger(kernel.CoreService).WithUser(webSrv)
	ucAuthenticate := usecase.NewAuthenticate(authLog, authManager, authManager, boxManager)
	ucCreateZettel := usecase.NewCreateZettel(ucLog, rtConfig, protectedBoxManager)
	ucGetMeta := usecase.NewGetMeta(protectedBoxManager)
	ucGetAllMeta := usecase.NewGetAllMeta(protectedBoxManager)
	ucGetZettel := usecase.NewGetZettel(protectedBoxManager)
	ucParseZettel := usecase.NewParseZettel(rtConfig, ucGetZettel)
	ucEvaluate := usecase.NewEvaluate(rtConfig, ucGetZettel, ucGetMeta)
	ucListMeta := usecase.NewListMeta(protectedBoxManager)
	ucListRoles := usecase.NewListRole(protectedBoxManager)
	ucListTags := usecase.NewListTags(protectedBoxManager)
	ucZettelContext := usecase.NewZettelContext(protectedBoxManager, rtConfig)
	ucDelete := usecase.NewDeleteZettel(ucLog, protectedBoxManager)
	ucUpdate := usecase.NewUpdateZettel(ucLog, protectedBoxManager)
	ucRename := usecase.NewRenameZettel(ucLog, protectedBoxManager)
	ucUnlinkedRefs := usecase.NewUnlinkedReferences(protectedBoxManager, rtConfig)
	ucRefresh := usecase.NewRefresh(ucLog, protectedBoxManager)

	webSrv.Handle("/", wui.MakeGetRootHandler(protectedBoxManager))

	// Web user interface
	if !authManager.IsReadonly() {
		webSrv.AddZettelRoute('b', server.MethodGet, wui.MakeGetRenameZettelHandler(
			ucGetMeta, &ucEvaluate))
		webSrv.AddZettelRoute('b', server.MethodPost, wui.MakePostRenameZettelHandler(&ucRename))
		webSrv.AddZettelRoute('c', server.MethodGet, wui.MakeGetCopyZettelHandler(
			ucGetZettel, usecase.NewCopyZettel()))
		webSrv.AddZettelRoute('c', server.MethodPost, wui.MakePostCreateZettelHandler(&ucCreateZettel))
		webSrv.AddZettelRoute('d', server.MethodGet, wui.MakeGetDeleteZettelHandler(
			ucGetMeta, ucGetAllMeta, &ucEvaluate))
		webSrv.AddZettelRoute('d', server.MethodPost, wui.MakePostDeleteZettelHandler(&ucDelete))
		webSrv.AddZettelRoute('e', server.MethodGet, wui.MakeEditGetZettelHandler(ucGetZettel))
		webSrv.AddZettelRoute('e', server.MethodPost, wui.MakeEditSetZettelHandler(&ucUpdate))
		webSrv.AddZettelRoute('f', server.MethodGet, wui.MakeGetFolgeZettelHandler(
			ucGetZettel, usecase.NewFolgeZettel(rtConfig)))
		webSrv.AddZettelRoute('f', server.MethodPost, wui.MakePostCreateZettelHandler(&ucCreateZettel))
		webSrv.AddZettelRoute('g', server.MethodGet, wui.MakeGetNewZettelHandler(
			ucGetZettel, usecase.NewNewZettel()))
		webSrv.AddZettelRoute('g', server.MethodPost, wui.MakePostCreateZettelHandler(&ucCreateZettel))
	}
	webSrv.AddListRoute('f', server.MethodGet, wui.MakeSearchHandler(
		usecase.NewSearch(protectedBoxManager), &ucEvaluate))
	webSrv.AddListRoute('g', server.MethodGet, wui.MakeGetGoActionHandler(&ucRefresh))
	webSrv.AddListRoute('h', server.MethodGet, wui.MakeListHTMLMetaHandler(
		ucListMeta, ucListRoles, ucListTags, &ucEvaluate))
	webSrv.AddZettelRoute('h', server.MethodGet, wui.MakeGetHTMLZettelHandler(
		&ucEvaluate, ucGetMeta))
	webSrv.AddListRoute('i', server.MethodGet, wui.MakeGetLoginOutHandler())
	webSrv.AddListRoute('i', server.MethodPost, wui.MakePostLoginHandler(&ucAuthenticate))
	webSrv.AddZettelRoute('i', server.MethodGet, wui.MakeGetInfoHandler(
		ucParseZettel, &ucEvaluate, ucGetMeta, ucGetAllMeta, ucUnlinkedRefs))
	webSrv.AddZettelRoute('k', server.MethodGet, wui.MakeZettelContextHandler(
		ucZettelContext, &ucEvaluate))

	// API
	webSrv.AddListRoute('a', server.MethodPost, a.MakePostLoginHandler(&ucAuthenticate))
	webSrv.AddListRoute('a', server.MethodPut, a.MakeRenewAuthHandler())
	webSrv.AddListRoute('j', server.MethodGet, a.MakeListMetaHandler(ucListMeta))
	webSrv.AddZettelRoute('j', server.MethodGet, a.MakeGetZettelHandler(ucGetZettel))
	webSrv.AddZettelRoute('l', server.MethodGet, a.MakeGetLinksHandler(ucEvaluate))
	webSrv.AddZettelRoute('m', server.MethodGet, a.MakeGetMetaHandler(ucGetMeta))
	webSrv.AddZettelRoute('o', server.MethodGet, a.MakeGetOrderHandler(
		usecase.NewZettelOrder(protectedBoxManager, ucEvaluate)))
	webSrv.AddZettelRoute('p', server.MethodGet, a.MakeGetParsedZettelHandler(ucParseZettel))
	webSrv.AddListRoute('r', server.MethodGet, a.MakeListRoleHandler(ucListRoles))
	webSrv.AddListRoute('t', server.MethodGet, a.MakeListTagsHandler(ucListTags))
	webSrv.AddZettelRoute('u', server.MethodGet, a.MakeListUnlinkedMetaHandler(
		ucGetMeta, ucUnlinkedRefs, &ucEvaluate))
	webSrv.AddListRoute('v', server.MethodPost, a.MakePostEncodeInlinesHandler(ucEvaluate))
	webSrv.AddZettelRoute('v', server.MethodGet, a.MakeGetEvalZettelHandler(ucEvaluate))
	webSrv.AddZettelRoute('x', server.MethodGet, a.MakeZettelContextHandler(ucZettelContext))
	webSrv.AddListRoute('z', server.MethodGet, a.MakeListPlainHandler(ucListMeta))
	webSrv.AddZettelRoute('z', server.MethodGet, a.MakeGetPlainZettelHandler(ucGetZettel))
	if !authManager.IsReadonly() {
		webSrv.AddListRoute('j', server.MethodPost, a.MakePostCreateZettelHandler(&ucCreateZettel))
		webSrv.AddZettelRoute('j', server.MethodPut, a.MakeUpdateZettelHandler(&ucUpdate))
		webSrv.AddZettelRoute('j', server.MethodDelete, a.MakeDeleteZettelHandler(&ucDelete))
		webSrv.AddZettelRoute('j', server.MethodMove, a.MakeRenameZettelHandler(&ucRename))
		webSrv.AddListRoute('z', server.MethodPost, a.MakePostCreatePlainZettelHandler(&ucCreateZettel))
		webSrv.AddZettelRoute('z', server.MethodPut, a.MakeUpdatePlainZettelHandler(&ucUpdate))
		webSrv.AddZettelRoute('z', server.MethodDelete, a.MakeDeleteZettelHandler(&ucDelete))
		webSrv.AddZettelRoute('z', server.MethodMove, a.MakeRenameZettelHandler(&ucRename))
	}

	if authManager.WithAuth() {
		webSrv.SetUserRetriever(usecase.NewGetUserByZid(boxManager))
	}
}
