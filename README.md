# TinyTinyRSS Tool
`tt` makes it easy to view and manipulate the feeds your Tiny Tiny RSS account
is subscribed to. It communicates with your tt-rss server using the
[tt-rss API](tt-rss.org/redmine/projects/tt-rss/wiki/JsonApiReference).

`tt` was developed by [Jeremy W. Sherman](https://jeremywsherman.com) and lives
at [jeremy-w/tt](https://github.com/jeremy-w/tt).

`tt` goes great with [Newsbeuter](http://newsbeuter.org/).

## Basic Idea
- `tt ls` lists categories (suffixed with `/`) and feeds without categories.
- `tt ls /category` lists subcategories and feeds in that category.
- `tt ls -R` will recursively list categories and feeds from that point down.
- `tt cp feedurl /category` adds the feed to the category. (Mnemonic: "copy".)
- `tt rm feedID` removes the feed with the specified ID from your subscriptions.
