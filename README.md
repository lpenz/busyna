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


