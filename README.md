# TinyTinyRSS Tool
`ttrss-tool` makes it easy to view and manipulate the feeds your Tiny Tiny RSS
account is subscribed to. It communicates with your tt-rss server using the
[tt-rss API](tt-rss.org/redmine/projects/tt-rss/wiki/JsonApiReference).

`ttrss-tool` was developed by [Jeremy W. Sherman](https://jeremywsherman.com)
and lives at [jeremy-w/tt](https://github.com/jeremy-w/tt).

`ttrss-tool` goes great with [Newsbeuter](http://newsbeuter.org/).

## Basic Idea
- `ttrss-tool ls`
  lists categories (suffixed with `/`) and feeds without categories.
- `ttrss-tool ls /category`
  lists subcategories and feeds in that category.
- `ttrss-tool ls -R`
  recursively lists categories and feeds.
- `ttrss-tool cp feedurl /category`
  adds the feed to the category. (Mnemonic: "copy".)
- `ttrss-tool rm feedID`
  removes the feed with the specified ID from your subscriptions.
