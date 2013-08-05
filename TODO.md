# TODO
- User should be able to list categories and feeds.
- User should be able to unsubscribe from a feed.
- User should be able to subscribe to a feed under a specified category.
- User should be able to move a feed under a category.
  - Can always unsubscribe then resubscribe in that location, if the API lacks
    a "move feed" equivalent.
- User should be able to add a category.
  - API might be lacking entirely here.
- User should be able to recursively list categories and feeds.
  - We could be smarter, but a first pass should just recursively call our
    non-recursive list function.

# DONE
- User should be able to subscribe to a feed.
  [completed 2013-08-04T00:31:39Z-0400]
- User should be able to store connection info in a dotfile.
  [completed 2013-08-04T03:17:20Z-0400]
