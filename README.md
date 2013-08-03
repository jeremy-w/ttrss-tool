# TinyTinyRSS Tool
`ttrss-tool` makes it easy to view and manipulate the feeds your Tiny Tiny RSS
account is subscribed to. It communicates with your tt-rss server using the
[tt-rss API](http://tt-rss.org/redmine/projects/tt-rss/wiki/JsonApiReference).

`ttrss-tool` was developed by [Jeremy W. Sherman](https://jeremywsherman.com)
and lives at [jeremy-w/ttrss-tool](https://github.com/jeremy-w/ttrss-tool).

`ttrss-tool` goes great with [Newsbeuter](http://newsbeuter.org/).
(It seemed a shame to have to hit the Web to edit your feedlist when using
ttrss as the backend, and thus was `ttrss-tool` born.)

## Usage
- `ttrss-tool ls` (short for `ls /`)
  lists top-level categories and uncategorized feeds.
- `ttrss-tool ls /category`
  lists the categories and feeds directly under that category.
- `ttrss-tool ls -R`
  recursively lists categories and feeds.
- `ttrss-tool mkdir /category`
  adds a new, empty (sub)category.
- `ttrss-tool ln feedurl /category`
  links a new feed into the specified category.
- `ttrss-tool rm feed`
  removes the feed with the specified ID from your subscriptions.

## Authentication
Currently, `ttrss-tool` requires three flags on all invocations:

- `-u,--user`: the user name
- `-p,--pass`: the password
- `-a,--addr`: the address of your ttrss instance, like
  `https://example.org/ttrss`

These are flags for now, but are likely to migrate into a dotfile sooner
rather than later, because whoah too much boring prefatory typing.

## Printing Categories and Feeds
**TODO:** Describe how feeds and categories are displayed, and what the fields
mean.
