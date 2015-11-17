[![Build Status](https://travis-ci.org/lpenz/busyna.png?branch=master)](https://travis-ci.org/lpenz/busyna) 
[![Coverage Status](https://coveralls.io/repos/lpenz/busyna/badge.png?branch=master)](https://coveralls.io/r/lpenz/busyna?branch=master)


# busyna

BUild SYstems Never Alone - an alternative to crafting build system configuration by hand

A quick summary for the impatient:
```
busyna.rc: shell-like "script"
   |
   | run: trace commands, discover targets and dependencies
   v
busyna.db
   |
   | deploy: create the build file for the target build system
   v
Makefile/SConstruct/etc.
```


## The busyna.rc file

The `busyna.rc` file has all the commands to build something, in the order they
should be executed. The syntax is a subset of shell - and you can even use
bash to run it to debug.

But the real functionality comes when you *busyna run* it.


## busyna run

When you *busyna run* the `busyna.rc` file, though, all operations that read or
write to files are traced, and the files used collected
as *dependencies* (read files) and *targets* (written files). That information
is stored in a `busyna.db` file, that can later be used to generate
a very precise *Makefile*.


## busyna deploy

After obtaining the `busyna.db` file, you can use *busyna deploy* to create
the corresponding *Makefile* (support for other build systems will be added).
You can use this file as your runtime builds system, to bootstrap one or or to
check if the dependencies of an existing *Makefile* are complete.

The important thing is: your are not alone maintaining the dependencies
anymore - you can just let the computer do that for you.


