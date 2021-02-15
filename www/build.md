# How to build the Zettelstore
## Prerequisites
You must install the following software:

* A current, supported [release of Go](https://golang.org/doc/devel/release.html),
* [golint](https://github.com/golang/lint|golint),
* [Fossil](https://fossil-scm.org/).

## Clone the repository
Most of this is covered by the excellent Fossil documentation.

1. Create a directory to store your Fossil repositories.
   Let's assume, you have created <tt>$HOME/fossil</tt>.
1. Clone the repository: `fossil clone https://zettelstore.de/ $HOME/fossil/zettelstore.fossil`.
1. Create a working directory.
   Let's assume, you have created <tt>$HOME/zettelstore</tt>.
1. Change into this directory: `cd $HOME/zettelstore`.
1. Open development: `fossil open $HOME/fossil/zettelstore.fossil`.

(If you are not able to use Fossil, you could try the Git mirror
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

`COMMAND` is one of:

* `build`: builds the software with correct version information and places it
  into a freshly created directory <tt>bin</tt>.
* `check`: checks the current state of the working directory to be ready for
  release (or commit).
* `release`: executes `check` command and if this was successful, builds the
  software for various platforms, and creates ZIP files for each executable.
  Everything is placed in the directory <tt>releases</tt>.
* `clean`: removes the directories <tt>bin</tt> and <tt>releases</tt>.
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
