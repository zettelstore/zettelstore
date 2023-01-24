# How to build Zettelstore
## Prerequisites
You must install the following software:

* A current, supported [release of Go](https://go.dev/doc/devel/release),
* [staticcheck](https://staticcheck.io/),
* [shadow](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/shadow),
* [unparam](https://mvdan.cc/unparam),
* [Fossil](https://fossil-scm.org/),
* [Git](https://git-scm.org) (so that Go can download some dependencies).

See folder <tt>docs/development</tt> (a zettel box) for details.

## Clone the repository
Most of this is covered by the excellent Fossil [documentation](https://fossil-scm.org/home/doc/trunk/www/quickstart.wiki).

1. Create a directory to store your Fossil repositories.
   Let's assume, you have created <tt>$HOME/fossils</tt>.
1. Clone the repository: `fossil clone https://zettelstore.de/ $HOME/fossils/zettelstore.fossil`.
1. Create a working directory.
   Let's assume, you have created <tt>$HOME/zettelstore</tt>.
1. Change into this directory: `cd $HOME/zettelstore`.
1. Open development: `fossil open $HOME/fossils/zettelstore.fossil`.

(If you are not able to use Fossil, you could try the GitHub mirror
<https://github.com/zettelstore/zettelstore>.)

## The build tool
In directory <tt>tools</tt> there is a Go file called <tt>build.go</tt>.
It automates most aspects, (hopefully) platform-independent.

The script is called as:

```
go run tools/build.go [-v] COMMAND
```

The flag `-v` enables the verbose mode.
It outputs all commands called by the tool.

Some important `COMMAND`s are:

* `build`: builds the software with correct version information and puts it
  into a freshly created directory <tt>bin</tt>.
* `check`: checks the current state of the working directory to be ready for
  release (or commit).
* `clean`: removes the build directories and cleans the Go cache.
* `version`: prints the current version information.

Therefore, the easiest way to build your own version of the Zettelstore
software is to execute the command

```
go run tools/build.go build
```

In case of errors, please send the output of the verbose execution:

```
go run tools/build.go -v build
```
