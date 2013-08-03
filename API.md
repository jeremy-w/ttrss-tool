# TinyTinyRSS API
You'll find scant and confusing documentation at the
[official website](https://github.com/gothfox/Tiny-Tiny-RSS).

Note in particular that you have no idea what format the data you get back is
in, and how to interpret it. That's what drove me to viewing the source.

So, skip the API docs. No-one writes them, and no-one should be forced to read
them.

Instead, grab the latest tarball.

Of interest:

- `api/index.php`: Barest stub that routes the API calls to the API class.
- `classes/api.php`: Wondered why the API operations had kind of funky
  capitalization? It's because they're just method names on the `API` class.
  Hello RPC!

It's really the `api.php` file you'll want to look over. This is where you can
parse out operations (all the methods), parameters (`$_REQUEST["param"]`), and
what each call does and returns.

You'll find some surprises. For example, as of 1.9, `$seq` is NULL when you
forget to supply a session key, but `API::wrap` still tries to set it, and you
get a bogus value back for that key in the JSON response.

## Intended Use of the API
The official docs do scatter some guidance throughout:

- `getCategories`: Intended starting point.

  With `enableNested`, you get the top-level categories; without it, you get all but empty categories (there's another flag for those).

  Once you have a category, you can drill into it using `getFeeds`.
- `getFeeds`: You provide the `cat_id`, it spits back feed objects.

  There are some special category IDs (including more than are listed in the docs).
  These are all represented by negative ID values. Not even the source code uses
  names for these categories, just the raw numbers.

  The only immediately interesting one is `0`, the Uncategorized feeds.

## API Used by ttrss-tool
We only need to use a fraction of the odd API. What follows is valid as of
the 1.9 release; you can treat it as documentation for the portion of the API
we work with. No attempt was made to be exhaustive: we've got a job to do here!

## APIs Used by Subcommand
### Ln
Uses `subscribeToFeed`, naturally enough.

We have to convert the category name to the corresponding category ID using
`getCategories`.

### Ls
Listing categories and feeds can use one of several API calls.
The most general-purpose is `getFeedTree`; its sole drawback is lack of
counters (unread status, number of items, etc.); that, and the mess of data
it spits back that you then have to dig through.

### Mkdir
Uh, looks like you can't actually create a category via the stock tt-rss
plugin API. Perhaps we can fork and PR, or, failing that, just distribute
a plugin.

## Schema
TinyTinyRSS is a very database-conscious app.
It creates raw query strings and sends them over to the DB, then parses them
out itself.

You'll want to look at the Postgres or MySQL schema in `schema/`.
There's a good chunk of seed data and magic numbers supplied via the DB.

## Random API
So, there's more API than just `api.php`.
There's a non-JSON, regular GET query-string–based API reached via
`public.php?op=NAME`.

There's a compatibility shim at `backend.php` that bridges through to the
corresponding `public.php` for certain operations:

- globalUpdateFeeds
- rss
- getUnread
- getProfiles
- share
- fbexport
- logout
- pubsub

For the rest, it does a general search of registered handlers responding to
the provided `op` and `subop`/`method` pair.

The rest of the public operations can be found by searching for
`add_handler("public"`, which registers a handler for a given op under
`public.php`.
This is where `fbexport` gets provided, by the `init` plugin.

The bulk of the operations are exposed as methods on
`classes/handler/public.php`.

## Miscellaneous Notes
- Plugins can provide their own API by calling `PluginHost::add_api_method`,
  and clients can then provide those API calls as `op`. So long as it doesn't
  shadow tt-rss–provided API, `api/index.php` will fall through to it.
- The actual API methods just do marshaling. The real work of method `doBlah`
  happens in the corresponding method `apiDoBlah`.
- `fix_url` will rewrite `feed:` to `http:`, add the `http:` scheme if none was
  provided, and append a `/` if the user gave a raw domain.
  This is used when the user provides a subscription URL.
- The server will grunge through HTML to find a feed link; if it finds multiple
  feed links (very commonly, you'll find both an articles feed and a comments
  feed), it will give up and make the user pick one.
