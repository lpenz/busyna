[![Build Status](https://travis-ci.org/lpenz/busyna.png?branch=master)](https://travis-ci.org/lpenz/busyna)
[![Coverage Status](https://coveralls.io/repos/lpenz/busyna/badge.png?branch=master)](https://coveralls.io/r/lpenz/busyna?branch=master)


# busyna

Tired of maintaining dependencies by hand on your build system? **busyna**
starts with a simple shell-like script that, when run, traces the files used by
each command. That information can then be used to create a correct and
optimized build system script (Makefile/SConstruct/etc.) that you can then use
at will.

In essence:
```
busyna.rc: shell-like busyna script
   |
   | run: execute and trace commands, discover dependencies and targets
   v
busyna.db: database with "run" information
   |
   | deploy: create the script for the target build system
   v
Makefile/graphviz/SConstruct/etc.
```

**busyna** can also be used to optimize an existing build system or convert it
to another by *extracting* the `busyna.rc` from the existing Makefile instead
of writting it by hand:
```
Makefile: existing build infrastructure
   |
   |
   v
busyna.rc:
   |
   v
  ...
```


## The busyna.rc file

The `busyna.rc` file has all commands in the order they should be executed. The
syntax is a subset of shell - and you can even debug it by using bash.

But the real functionality comes when you *busyna run* it.


## busyna run

When you *busyna run* the `busyna.rc` file, all operations that read or write
to files are traced, and the files used collected as *dependencies* (read
files) and *targets* (written files). That information is stored in
a `busyna.db` file, that can later be used to generate a very optimized and
precise *Makefile*.


## busyna deploy

After obtaining the `busyna.db` file, you can use *busyna deploy* to create
the corresponding *Makefile* (support for other build systems will be added).
You can use this file as your runtime builds system, to bootstrap one or or to
check if the dependencies of an existing *Makefile* are complete.

The important thing is: your are not alone maintaining the dependencies
anymore - you can just let the computer do that for you.


## busyna extract

Instead of writing the `busyna.rc` file, you can *busyna extract* it from an
existing *Makefile*. The dependencies detected by busyna will usually be more
complete and minimalist. You can use the results to:
- generate the graphviz output to see your build system dependencies graphically
- generate another, optimized Makefile
- bootstrap a conversion to another build system (not yet supported)

Complex *Makefiles* will not be correctly converted without modifications. More
details on the [Case studies](#case-studies)


## Roadmap

- busyna-sh: enable `busyna.rc` execution with
  [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix)).
- Fix bugs and finish supporting the current features with correct detection
  of environment and directories.
- Validate the implementation by building make-based software projects - the
  linux kernel, for instance.
- Support further target build systems:
  - [tup](http://gittup.org/tup/)
  - [scons](http://www.scons.org/)
  - [shake](http://shakebuild.com/)
- Deal with `busyna.db` updates: perhaps a mode where the deploy command
 crates a special build system file that interacts with busyna to trace the
 executed commands that update the dependencies?


# Case studies

In this section we convert the build system of some free software tools and
show advantages provided by busyna.


## Vim

[Vim](http://www.vim.org) is the famous text editor, successor to *vi*.

This case is used as a test, and the complete script is available
at *scripts/vim-test*.


### Extracting busyna.rc

To be able to use *busyna-extract*, we make the following changes
in *vim/src/Makefile* (all commands are supposed to be executed inside *vim/src*):
- remove the line that sets the shell, as *busyna-extract* uses a custom shell
  to process the *Makefile*:

  ```
  sed -i '/SHELL = \/bin\/sh/d' Makefile
  ```
- Set *QUOTESED* to the empty string; the variable sets up processing that is
  not needed with *busyna-extract*:

  ```
  sed -i 's/^#\s*QUOTESED\s*=.*/QUOTESED=cat/' Makefile
  ```
- Remove the *echo* comment that has parenthesis in it, as that sets up a subshell
  when *busyna-extract* tries to pass it to */bin/sh*:

  ```
  sed -i 's@\(The name of the makefile MUST .*\)(.*@\1@' Makefile
  ```

After changing the *Makefile*, you can run:
```shell
busyna-extract make ../../busyna.rc
```
to create a *busyna.rc* file in the directory above the clone.


### Using busyna.rc

You can now:
- Build vim, using the *busyna.rc* file as a shell script:

  ```shell
  ../../busyna-rc
  ```
- Create the build database (used in further actions):

  ```
  busyna-run ../../busyna.rc busyna.db
  ```
- See the graphviz dependency graph (requires busyna.db):

  ```
  busyna-deploy dot busyna.db deps.dot
  dot -Tpdf -o deps.pdf deps.dot
  see deps.pdf
  ```
- Create a new *Makefile*, with complete minimal dependencies, and use it to
  build vim (requires busyna.db):

  ```
  busyna-deploy make busyna.db Makefile.busyna
  make -f Makefile.busyna clean
  make -f Makefile.busyna
  ```
  Try changing a source file and re-running make, to see that the dependencies
  are actually correct.

