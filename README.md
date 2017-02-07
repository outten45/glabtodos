# glabtodos

A command line tool to notify you have pending todos from a gitlab instance on
your Mac. There are 2 types of notifications.

* If you run [AnyBar](https://github.com/tonsky/AnyBar), it will turn the dot
  Red when you have pending todos.
* Using [notificator](https://github.com/0xAX/notificator) you will get a popup
  notification if you have any pending todos.

# Setup

Add the following environmental variables:

* `GLAB_HOST` - The schema and host (for example https://gitlab.example.com)
* `GLAB_APIPATH` - The URI of the API (for example /api/v3/)
* `GLAB_TOKEN` - Your access token setup in GLAB_LAB
* `GLAB_DELAY` - The display between polling and defaults to 90s (90 seconds)



