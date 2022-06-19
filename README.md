# khinsider-sqlite3

> A small Go program that scrapes downloads.khinsider.com into a sqlite3 db

Eventually intended to replace [khinsider-index](https://github.com/marcus-crane/khinsider-index).

The goal is to prepopulate a sqlite3 DB with information to allow dynamic queries and full searching client side, in order to greatly speed up [khinsider](https://github.com/marcus-crane/khinsider) which is currently bound, in terms of performance, by having to touch multiple webpages.

It has an index of what albums exist but it still needs to build up information about them whereas this repo would be intended to prepopulate a sqlite3 DB with all of that metadata in advance.
