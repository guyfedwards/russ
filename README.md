# nom
> Feed me

`nom` is a terminal based RRS feed reader using [Glow](https://github.com/charmbracelet/glow) styled markdown to improve the reading experience and a simple TUI using [Bubbletea](https://github.com/charmbracelet/bubbletea).
- Local sync and offline reading
- Backend connections (miniflux, freshrss supported)
- Vim style keybindings for navigation
- Plenty more features such as mark read/unread, filtering and feed naming

![](./.github/demo.gif)

## Install
See [releases](https://github.com/guyfedwards/nom/releases) for binaries. E.g.
```sh
$ curl -L https://github.com/guyfedwards/nom/releases/download/v2.1.4/nom_2.1.4_darwin_amd64.tar.gz | tar -xzvf -
```

## Config
Config lives by default in `$XDG_CONFIG_HOME/nom/config.yml`
### Feeds
Add feeds with the `add` command 
```sh
$ nom add <url>
```
or add directly to the config at `$XDG_CONFIG_HOME/nom/config.yml` on unix systems and `$HOME/Library/Application Support/nom/config.yml` on darwin.
```yaml
feeds:
  - url: https://dropbox.tech/feed
    # name will be prefixed to all entries in the list
    name: dropbox 
  - url: https://snyk.io/blog/feed
```
You can customise the location of the config file with the `--config-path` flag.

### Show read (default: false)
Show read items by default. (can be toggled with M)
```yaml
showread: true
```
### Auto read (default: false)
Automatically mark items as read on selection or navigation through items. ()
```yaml
autoread: true
```

### Backends
As well as adding feeds directly, you can pull in feeds from another source. You can add multiple backends and the feeds will all be added.
```yaml
backends:
  miniflux:
    host: http://myminiflux.foo
    api_key: jafksdljfladjfk
  freshrss:
    host: http://myfreshrss.bar
    user: admin
    password: muchstrong
```

## Store
`nom` uses sqlite as a store for feeds and metadata. It is stored next to the config in `$XDG_CONFIG_HOME/nom/nom.db`. This can be backed up like any file and will store articles, read state etc. It can also be deleted to start from scratch redownloading all articles and no state.

## Filtering
Within the `nom` view, you can filter by title pressing the `/` character. Filters can be applied easily. Here's some examples:
- `f:my_feed feed:my_second_feed` - matches `my_feed` and `my_second_feed`
- `feedname:"my feed - with spaces"` - matches `my feed - with spaces`
- `feed:'my feed, with single quotes!'` - matches `my feed, with single quotes!`
- `feed:my\ feed\ with\ escaped\ spaces!` - matches `my feed with escaped spaces!`

More filters to be added soon!

## Building and Running via Docker
Build nom image
```sh
docker build -t nom .
```
This embeds the local docker-config.yml file into the container and will be used by default.

Running the nom via docker
```sh
docker run --rm -it nom
```
Use the `-v` command line argument to mount a local config onto `/app/docker-config.yml` as desired.


## Dev setup
You can use the `backends-compose.yml` to spin up a local instance of miniflux and freshrss if needed for development.

```sh
$ docker-compose -f backends-compose.yml up
```
