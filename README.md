# TinyTinyRSS Tool
`ttrss-tool` makes it easy to view and manipulate the feeds your Tiny Tiny RSS
account is subscribed to. It communicates with your tt-rss server using the
[tt-rss API](http://tt-rss.org/redmine/projects/tt-rss/wiki/JsonApiReference).

`ttrss-tool` was developed by [Jeremy W. Sherman](https://jeremywsherman.com)
and lives at [jeremy-w/ttrss-tool](https://github.com/jeremy-w/ttrss-tool).

`ttrss-tool` goes great with [Newsbeuter](http://newsbeuter.org/).

## Basic Idea
- `ttrss-tool ls`
  lists categories (suffixed with '/'), special categories (suffixed with '%'),
  and feeds (no suffix) and virtual feeds (suffixed with '@') at that level.
- `ttrss-tool ls /category`
  lists subcategories and feeds in that category.
- `ttrss-tool ls -R`
  recursively lists categories and feeds.
- `ttrss-tool mkdir /category`
  adds a new, empty (sub)category.
- `ttrss-tool ln feedurl /category`
  links a new feed into the specified category.
- `ttrss-tool rm feedID`
  removes the feed with the specified ID from your subscriptions.
