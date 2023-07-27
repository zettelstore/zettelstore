# How to build Zettelstore
## Prerequisites
You must install the following software:

* A current, supported [release of Go](https://go.dev/doc/devel/release),
* [staticcheck](https://staticcheck.io/),
* [shadow](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/shadow),
* [unparam](https://mvdan.cc/unparam),
* [govulncheck](https://golang.org/x/vuln/cmd/govulncheck),
* [Fossil](https://fossil-scm.org/),
* [Git](https://git-scm.org) (so that Go can download some dependencies).

See folder `docs/development` (a zettel box) for details.

## Clone the repository
Most of this is covered by the excellent Fossil [documentation](https://fossil-scm.org/home/doc/trunk/www/quickstart.wiki).

1. Create a directory to store your Fossil repositories.
   Let's assume, you have created `$HOME/fossils`.
1. Clone the repository: `fossil clone https://zettelstore.de/ $HOME/fossils/zettelstore.fossil`.
1. Create a working directory.
   Let's assume, you have created `$HOME/zettelstore`.
1. Change into this directory: `cd $HOME/zettelstore`.
1. Open development: `fossil open $HOME/fossils/zettelstore.fossil`.

## The build tool
In the directory `tools` there is a Go file called `build.go`.
It automates most aspects, (hopefully) platform-independent.

The script is called as:

```
go run tools/build.go [-v] COMMAND
```

The flag `-v` enables the verbose mode.
It outputs all commands called by the tool.

Some important `COMMAND`s are:

* `build`: builds the software with correct version information and puts it
  into a freshly created directory `bin`.
* `check`: checks the current state of the working directory to be ready for
  release (or commit).
* `clean`: removes the build directories and cleans the Go cache.
* `version`: prints the current version information.
* `tools`: installs / updates the tools described above: staticcheck, shadow,
  unparam, govulncheck.

Therefore, the easiest way to build your own version of the Zettelstore
software is to execute the command

```
go run tools/build.go build
```

In case of errors, please send the output of the verbose execution:

```
go run tools/build.go -v build
```

## A note on the use of Fossil
Zettelstore is managed by the Fossil version control system.
Fossil is an alternative to the ubiquitous Git version control system.
However, Go seems to prefer Git and popular platforms that just support Git.

Some dependencies of Zettelstore, namely [Zettelstore client](https://zettelstore.de/client) and [sx](https://zettelstore.de/sx), are also managed by Fossil.
Depending on your development setup, some error messages might occur.

If the error message mentions an environment variable called `GOVCS` you should set it to the value `GOVCS=zezzelstore.de:fossil` (alternatively more generous to `GOVCS=*:all`).
Since the Go build system is coupled with Git and some special platforms, you allow ot to download a Fossil repository from the host `zettelstore.de`.
The build tool set `GOVCS` to the right value, but you may use other `go` commands that try to download a Fossil repository.

On some operating systems, namely Termux on Android, an error message might state that an user cannot be determined (`cannot determine user`).
In this case, Fossil is allowed to download the repository, but cannot associate it with an user name.
Set the environment variable `USER` to any user name, like: `USER=nobody go run tools/build.go build`.