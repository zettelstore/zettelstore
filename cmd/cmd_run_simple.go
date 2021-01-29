//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/place"
	"zettelstore.de/z/web/server"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func flgSimpleRun(fs *flag.FlagSet) {
	fs.String("d", "", "zettel directory")
}

func runSimpleFunc(*flag.FlagSet) (int, error) {
	p := startup.PlaceManager()
	if _, err := p.GetMeta(context.Background(), id.WelcomeZid); err != nil {
		if err == place.ErrNotFound {
			updateWelcomeZettel(p)
		}
	}

	listenAddr := startup.ListenAddress()
	readonlyMode := startup.IsReadOnlyMode()
	logBeforeRun(listenAddr, readonlyMode)
	if idx := strings.LastIndexByte(listenAddr, ':'); idx >= 0 {
		log.Println()
		log.Println("--------------------------")
		log.Printf("Open your browser and enter the following URL:")
		log.Println()
		log.Printf("    http://localhost%v", listenAddr[idx:])
	}

	handler := setupRouting(startup.PlaceManager(), readonlyMode)
	srv := server.New(listenAddr, handler)
	if err := srv.Run(); err != nil {
		return 1, err
	}
	return 0, nil
}

// runSimple is called, when the user just starts the software via a double click
// or via a simple call ``./zettelstore`` on the command line.
func runSimple() {
	dir := "./zettel"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create zettel directory %q (%s)\n", dir, err)
		os.Exit(1)
	}
	executeCommand("run-simple", "-d", dir)
}

func updateWelcomeZettel(p place.Place) {
	m := meta.New(id.WelcomeZid)
	m.Set(meta.KeyTitle, "Welcome to Zettelstore")
	m.Set(meta.KeyRole, meta.ValueRoleZettel)
	m.Set(meta.KeySyntax, meta.ValueSyntaxZmk)
	zid, err := p.CreateZettel(
		context.Background(),
		domain.Zettel{Meta: m, Content: domain.NewContent(welcomeZettelContent)},
	)
	if err == nil {
		p.RenameZettel(context.Background(), zid, id.WelcomeZid)
	}
}

var welcomeZettelContent = `Thank you for using Zettelstore!

You will find the lastest information about Zettelstore at [[https://zettelstore.de]].
Check that website regulary for [[upgrades|https://zettelstore.de/home/doc/trunk/www/download.wiki]] to the latest version.
You should consult the [[change log|https://zettelstore.de/home/doc/trunk/www/changes.wiki]] before upgrading.
Sometimes, you have to edit some of your Zettelstore-related zettel before upgrading.
Since Zettelstore is currently in a development state, every upgrade might fix some of your problems.
To check for versions, there is a zettel with the [[current version|00000000000001]] of your Zettelstore.

If you have problems concerning Zettelstore,
do not hesitate to get in [[contact with the main developer|mailto:ds@zettelstore.de]].

=== Reporting errors
If you have encountered an error, please include the content of the following zettel in your mail:
* [[Zettelstore Version|00000000000001]]
* [[Zettelstore Operating System|00000000000003]]
* [[Zettelstore Startup Configuration|00000000000096]]
* [[Zettelstore Startup Values|00000000000098]]
* [[Zettelstore Runtime Configuration|00000000000100]]

Additionally, you have to describe, what you have done before that error occurs
and what you have expected instead.
Please do not forget to include the error message, if there is one.

Some of above Zettelstore zettel can only be retrieved if you enabled ""expert mode"".
Otherwise, only some zettel are linked.
To enable expert mode, edit the zettel [[Zettelstore Runtime Configuration|00000000000100]]:
please set the metadata value of the key ''expert-mode'' to true.
To do you, enter the string ''expert-mode:true'' inside the edit view of the metadata.

=== Information about this zettel
This zettel was generated automatically.
Every time you start Zettelstore by double clicking in your graphical user interface,
or by just starting it in a command line via something like ''zettelstore'', and this zettel
does not exist, it will be generated.
This allows you to edit this zettel for your own needs.

If you don't need it anymore, you can delete this zettel by clicking on ""Info"" and then
on ""Delete"".
However, by starting Zettelstore as described above, the original version of this zettel
will be restored.

If you start Zettelstore with the ''run'' command, e.g. as a service or via command line,
this zettel will not be generated.
But if it exists before, it will not be deleted.
In this case, Zettelstore assumes that you have enough knowledge and that you do not need
zettel.
`
