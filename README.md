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

## Usage
```sh
Usage:
  nom [OPTIONS] [command]

Application Options:
  -v, --verbose      Show verbose logging
  -n, --number=      Number of results to show
  -p, --pager=       Pager to use for longer output. Set to false for no pager
      --config-path= Location of config.yml
  -f, --feed=        Feed(s) URL(s) for preview

Help Options:
  -h, --help         Show this help message

Available commands:
  add      Add feed
  config   Show config
  list     List feeds
  refresh  Refresh feeds
  version  Show Vesion
```

## Config
Config lives by default in `$XDG_CONFIG_HOME/nom/config.yml` or `$HOME/Library/Application Support/nom/config.yml` on darwin.  
You can customise the location of the config file with the `--config-path` flag.

### Feeds
Feeds are added to the config file and have a url and name.
```yaml
feeds:
  - url: https://dropbox.tech/feed
    # name will be prefixed to all entries in the list
    name: dropbox 
  - url: https://snyk.io/blog/feed
```
You can also add feeds with the `add` command:
```sh
$ nom add <url>
```
Feeds are editable within `nom` by pressing `E` to open the config in your `$EDITOR` or `$NOMEDITOR`. After editing feeds, you will need to then refresh with `r`.

#### Youtube feeds
To add youtube feeds you can go to a channel and run the following in the browser console to get the rss feed link:
```js
console.log(`https://www.youtube.com/feeds/videos.xml?channel_id=${document.querySelector("link[rel='canonical']").href.split('/channel/').reverse()[0]}`)
```

### Show read (default: false)
Show read items by default. (can be toggled with M)
```yaml
showread: true
```
### Auto read (default: false)
Automatically mark items as read on selection or navigation through items. 
```yaml
autoread: true
```
### Theme 
Theme allows some basic color overrides in the feed view and then setting a custom markdown render theme for the overall markdown view. `theme.glamour` can be one of "dark", "dracula", "light", "pink", "ascii" or "notty". See [here](https://github.com/charmbracelet/glamour/tree/master/styles/gallery) for previews and more info.
Colors can be hex or ASCII codes, they will be coerced depending on your terminal color settings.
```yaml
theme: 
  glamour: dark
  titleColor: "62"
  selectedItemColor: "170"
  filterColor: "#555555"
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

### Openers
By default links are opened in the browser, you can specify commands to open certain links based on a regex string.   
`regex` can be any valid golang regex string, it will be matched against the feed item link.  
`cmd` is run as a child command. The `%s` denotes the position of the link in the command.  
`takeover` dictates if the command should takeover the tty from nom. E.g. for opening links in lynx or other TUI.  
```yaml
openers:
  - regex: "youtube"
    cmd: "mpv %s"
  - regex: ".*"
    cmd: "lynx %s"
    takeover: true
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

### Debug logging
You can enable logging to a file using the `DEBUGNOM` env var. Passing any path will cause `log` calls to write there e.g. `DEBUGNOM=debug.log`
