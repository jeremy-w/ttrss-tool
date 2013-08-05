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

## Schema
TinyTinyRSS is a very database-conscious app.
It creates raw query strings and sends them over to the DB, then parses them
out itself.

You'll want to look at the Postgres or MySQL schema in `schema/`.
There's a good chunk of seed data and magic numbers supplied via the DB.

## API Used by ttrss-tool
We only need to use a fraction of the odd API. What follows is valid as of
the 1.9 release; you can treat it as documentation for the portion of the API
we work with. No attempt was made to be exhaustive: we've got a job to do here!

### GetFeedTree
The API represents a feed like so:

```js
{'auxcounter': 0,
 'bare_id': 5,
 'checkbox': False,
 'error': '',
 'icon': 'feed-icons/5.ico',
 'id': 'FEED:5',
 'name': 'Electrical',
 'param': '5:38',
 'type': 'feed',
 'unread': 0}
```

- `bare_id`: just `id` without the `FEED:` prefix
- `id`: just `bare_id` with `type` capitalized and prefixed to `:bare_id`; this
  is the ID in the feeds table in the DB
- `name`: initially set to `[Unknown]` on creation; updated when fetched
- `error`: set to the last error on the last update
- `param`: might be empty if the feed is provided by a plugin; otherwise, it's
  the local last-updated time
  - Date is formatted using the `SHORT_DATE_FORMAT` preference, which defaults
    to `'M d, G:i'`. Since user can change it, just pass it on through.
- `checkbox`: It's always false. No idea why the API thinks we need that
  non-info.
- `auxcounter`: This is always 0. Yeah.

A category looks like:

```js
{'auxcounter': 0,
 'bare_id': 0,
 'checkbox': False,
 'child_unread': 0,
 'id': 'CAT:0',
 'name': 'Uncategorized',
 'param': '(8 feeds)',
 'type': 'category',
 'unread': 0,
 'items': [/* child feeds/categories here */]}
```

- `type`, `bare_id`, `id`: like feed, but using `category` and `CAT`
- `param`: used to list how many feeds are in the category; might be localized
  and singular or plural, so just strip out the number if you care, or pass on
  through unmodified

### GetFeeds
Completely different encoding than `getFeedTree`:

- cat_id int if a feed, otherwise is_cat bool true
- title like name
- id like bare_id
- unread int

Categories will only be included if `include_nested` is set on the request.

Real feeds (not labels) also include:

- feed_url string
- has_icon bool
- last_updated int timestamp
- order_id int

Notice that, while this won't recurse for you, it gives you far better info.

Unfortunately, there is no way to get the root child categories from this API
call, because a zero `cat_id` is the default null Uncategorized category.

### GetCategories
Returns something different yet again:

- `id`
- `title`
- `unread`
- `order_id` int always zero for me, but maybe if you messed with the order in
  the Web UI, it would be different (not that we care)

## APIs Used by Subcommand
### CatPath
Many commands rely on a "catpath", a category path, like `/Cat1/Cat2/Cat3`.

The slow way to resolve this would be like so:

- Issue a `getCategories, include_empty: true, include_nested: true` to get the
  top-level categories and their IDs.
- Match name of first path component, grab the ID.
- Use that ID to issue a `getFeeds, include_nested: true`, and repeat from
  previous step.

A faster way:

- Issue a `getFeedTree, include_empty: true`
- The structure of the tree matches the category tree.

For feed URL, we still have to issue a `getFeeds`:

- If we use cat -3, we get all feeds, including virtual ones.
- Use -4 for all feeds, excluding virtual.
- Generally, we'll probably be able to specify precisely the feed we want.

tt-rss checks whether there's already a category with the same name under the same parent before adding another, so we shouldn't have to worry about categories with duplicate names at the same level.

It is possible to add a feed with the same name as a category.
We should pitch a fit if it actually causes us ambiguity and leave it at that,
though the user should be able to resolve the ambiguity by including
or excluding the final category-indicating slash on the catpath.

We might also just let them use the `CAT:1234` and `FEED:1234` syntax as
alternatives to the catpath.

### Ln
Uses `subscribeToFeed` and the `cat_id` found via CatPath, naturally enough.

### Ls
See "CatPath" section above.

One question is how to print things. `catName/` and `feedName`, then add a
long version that does `catName/ CAT:ID`
and `feedName FEED:ID lastUpdated error`.

### Mkdir
Uh, looks like you can't actually create a category via the stock tt-rss
plugin API. Perhaps we can fork and PR, or, failing that, just distribute
a plugin.

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

Plugins can register subops using `add_handler`.

Useful methods are exposed by the `rpc` op,
including most notably `quickAddCat cat: "title"`,
which at least lets you add categories at the root of the hierarchy.

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
- All API calls that take IDs want the bare ID, not the type-prefixed ID.
