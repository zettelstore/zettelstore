//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"context"
	"flag"
	"net/http"

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
	fs.String("c", "", "configuration file")
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
	protectedBoxManager, authPolicy := authManager.BoxWithPolicy(boxManager, rtConfig)
	kern := kernel.Main
	webLog := kern.GetLogger(kernel.WebService)

	var getUser getUserImpl
	logAuth := kern.GetLogger(kernel.AuthService)
	logUc := kern.GetLogger(kernel.CoreService).WithUser(&getUser)
	ucAuthenticate := usecase.NewAuthenticate(logAuth, authManager, authManager, boxManager)
	ucIsAuth := usecase.NewIsAuthenticated(logUc, &getUser, authManager)
	ucCreateZettel := usecase.NewCreateZettel(logUc, rtConfig, protectedBoxManager)
	ucGetMeta := usecase.NewGetMeta(protectedBoxManager)
	ucGetAllMeta := usecase.NewGetAllMeta(protectedBoxManager)
	ucGetZettel := usecase.NewGetZettel(protectedBoxManager)
	ucParseZettel := usecase.NewParseZettel(rtConfig, ucGetZettel)
	ucListMeta := usecase.NewListMeta(protectedBoxManager)
	ucEvaluate := usecase.NewEvaluate(rtConfig, ucGetZettel, ucGetMeta, ucListMeta)
	ucListSyntax := usecase.NewListSyntax(protectedBoxManager)
	ucListRoles := usecase.NewListRoles(protectedBoxManager)
	ucZettelContext := usecase.NewZettelContext(protectedBoxManager, rtConfig)
	ucDelete := usecase.NewDeleteZettel(logUc, protectedBoxManager)
	ucUpdate := usecase.NewUpdateZettel(logUc, protectedBoxManager)
	ucRename := usecase.NewRenameZettel(logUc, protectedBoxManager)
	ucUnlinkedRefs := usecase.NewUnlinkedReferences(protectedBoxManager, rtConfig)
	ucRefresh := usecase.NewRefresh(logUc, protectedBoxManager)
	ucVersion := usecase.NewVersion(kernel.Main.GetConfig(kernel.CoreService, kernel.CoreVersion).(string))

	a := api.New(
		webLog.Clone().Str("adapter", "api").Child(),
		webSrv, authManager, authManager, rtConfig, authPolicy)
	wui := webui.New(
		webLog.Clone().Str("adapter", "wui").Child(),
		webSrv, authManager, rtConfig, authManager, boxManager, authPolicy, &ucEvaluate)

	webSrv.Handle("/", wui.MakeGetRootHandler(protectedBoxManager))
	if assetDir := kern.GetConfig(kernel.WebService, kernel.WebAssetDir).(string); assetDir != "" {
		const assetPrefix = "/assets/"
		webSrv.Handle(assetPrefix, http.StripPrefix(assetPrefix, http.FileServer(http.Dir(assetDir))))
		webSrv.Handle("/favicon.ico", wui.MakeFaviconHandler(assetDir))
	}

	// Web user interface
	if !authManager.IsReadonly() {
		webSrv.AddZettelRoute('b', server.MethodGet, wui.MakeGetRenameZettelHandler(
			ucGetMeta, &ucEvaluate))
		webSrv.AddZettelRoute('b', server.MethodPost, wui.MakePostRenameZettelHandler(&ucRename))
		webSrv.AddZettelRoute('c', server.MethodGet, wui.MakeGetCreateZettelHandler(
			ucGetZettel, &ucCreateZettel, ucListRoles, ucListSyntax))
		webSrv.AddZettelRoute('c', server.MethodPost, wui.MakePostCreateZettelHandler(&ucCreateZettel))
		webSrv.AddZettelRoute('d', server.MethodGet, wui.MakeGetDeleteZettelHandler(
			ucGetMeta, ucGetAllMeta, &ucEvaluate))
		webSrv.AddZettelRoute('d', server.MethodPost, wui.MakePostDeleteZettelHandler(&ucDelete))
		webSrv.AddZettelRoute('e', server.MethodGet, wui.MakeEditGetZettelHandler(ucGetZettel, ucListRoles, ucListSyntax))
		webSrv.AddZettelRoute('e', server.MethodPost, wui.MakeEditSetZettelHandler(&ucUpdate))
	}
	webSrv.AddListRoute('g', server.MethodGet, wui.MakeGetGoActionHandler(&ucRefresh))
	webSrv.AddListRoute('h', server.MethodGet, wui.MakeListHTMLMetaHandler(ucListMeta, &ucEvaluate))
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
	webSrv.AddListRoute('j', server.MethodGet, a.MakeQueryHandler(ucListMeta))
	webSrv.AddZettelRoute('o', server.MethodGet, a.MakeGetOrderHandler(
		usecase.NewZettelOrder(protectedBoxManager, ucEvaluate)))
	webSrv.AddListRoute('q', server.MethodGet, a.MakeQueryHandler(ucListMeta))
	webSrv.AddZettelRoute('u', server.MethodGet, a.MakeListUnlinkedMetaHandler(
		ucGetMeta, ucUnlinkedRefs, &ucEvaluate))
	webSrv.AddListRoute('x', server.MethodGet, a.MakeGetDataHandler(ucVersion))
	webSrv.AddListRoute('x', server.MethodPost, a.MakePostCommandHandler(&ucIsAuth, &ucRefresh))
	webSrv.AddZettelRoute('x', server.MethodGet, a.MakeZettelContextHandler(ucZettelContext))
	webSrv.AddListRoute('z', server.MethodGet, a.MakeListPlainHandler(ucListMeta))
	webSrv.AddZettelRoute('z', server.MethodGet, a.MakeGetZettelHandler(ucGetMeta, ucGetZettel, ucParseZettel, ucEvaluate))
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

type getUserImpl struct{}

func (*getUserImpl) GetUser(ctx context.Context) *meta.Meta { return server.GetUser(ctx) }
