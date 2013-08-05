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
- `ttrss-tool ls [-lR] [catpath]`
  lists categories at `/` (default) or categories and feeds contained in
  the specified category.
- `ttrss-tool ln feed_url [catpath]`
  links a new feed into the specified category.
  If no category is specified, or `/` is specified, the feed is added to the
  default "Uncategorized" category.
- `ttrss-tool mkdir title`
  creates a new category.
  Due to API limitations, we can only create a top-level category.
- `ttrss-tool rm feed_spec`
  removes the specified feed.
  The feed can be specified by title using a catpath,
  by URL, or by numeric ID.
  (You can find the latter two bits of info using `ls -l`.)

## Authentication
`ttrss-tool` requires three pieces of information to operate:

- The address of your ttrss instance, such as `https://example.com/ttrss/`.
- Your account name on that instance.
- Your password.

You can supply this info by creating a config file `config` in a `ttrss-tool`
directory in `$XDG_CONFIG_HOME` (which defaults to `$HOME/.config`).

The config file should look like:

```json
{
  "addr": "https://example.com/ttrss/",
  "user": "alice",
  "pass": "keepoutmallory"
}
```

or

```json
{
  "addr": "https://example.com/ttrss/",
  "user": "alice"
}
```

With the latter form, you can either supply the password as a commandline flag,
or let `ttrss-tool` prompt you for it.

You can also supply this information as commandline flags:

- `-a,--addr`: the address of your ttrss instance, like
  `https://example.org/ttrss`
- `-u,--user`: the user name
- `-p,--pass`: the password

If both dotfile and commandline flags are present, then the flags win.

**NOTE:** The dotfile is just a JSON version of the long commandline flags.

## Printing Categories and Feeds
**TODO:** Describe how feeds and categories are displayed, and what the fields
mean.
