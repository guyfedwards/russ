package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/rss"
	"github.com/guyfedwards/nom/v2/internal/store"
)

type Commands struct {
	config *config.Config
	store  store.Store
}

func New(config *config.Config, store store.Store) Commands {
	return Commands{config, store}
}

func convertItems(its []store.Item) []list.Item {
	var items []list.Item

	for _, item := range its {
		items = append(items, ItemToTUIItem(item))
	}

	return items
}

func (c Commands) OpenLink(url string) tea.Cmd {
	for _, o := range c.config.Openers {
		match, err := regexp.MatchString(o.Regex, url)
		if err != nil {
			return tea.Quit
		}

		if match {
			c := fmt.Sprintf(o.Cmd, url)
			parts := strings.Fields(c)
			cmd := exec.Command(parts[0], parts[1:]...)

			if o.Takeover {
				return tea.ExecProcess(cmd, func(err error) tea.Msg {
					log.Println("OpenLink: takeover exec:", err)
					return nil
				})
			} else {
				if err := cmd.Run(); err != nil {
					log.Println("OpenLink: exec: ", err)
					return tea.Quit
				}
				return nil
			}
		}
	}

	c.OpenInBrowser(url)

	return nil
}

func (c Commands) OpenInBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		if IsWSL() {
			cmd = "cmd.exe"
			args = []string{"/c", "start"}
		} else {
			cmd = "xdg-open"
		}
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func IsWSL() bool {
	out, err := exec.Command("uname", "-a").Output()
	if err != nil {
		return false
	}
	// In some cases, uname on wsl outputs microsoft capitalized
	matched, _ := regexp.Match(`microsoft|Microsoft`, out)
	return matched
}

func IsWayland() bool {
	s := os.Getenv("XDG_SESSION_TYPE")
	return s == "wayland"
}

// Gets the subsystem host ip
// If the CLI is running under WSL the localhost url will not work so
// this function should return the real ip that we should redirect to
func GetWslHostName() string {
	out, err := exec.Command("wsl.exe", "hostname", "-I").Output()
	if err != nil {
		return "localhost"
	}
	return strings.TrimSpace(string(out))
}

func (c Commands) CleanFeeds() error {
	urls, err := c.store.GetAllFeedURLs()
	if err != nil {
		return fmt.Errorf("[commands.go]: %w", err)
	}

	var urlsToRemove []string

	for _, u := range urls {
		inFeeds := false
		for _, f := range c.config.Feeds {
			if f.URL == u {
				inFeeds = true
			}
		}

		if !inFeeds {
			urlsToRemove = append(urlsToRemove, u)
		}
	}

	for _, url := range urlsToRemove {
		err := c.store.DeleteByFeedURL(url, false)
		if err != nil {
			return fmt.Errorf("[commands.go]: %w", err)
		}
	}

	return nil
}

func (c Commands) TUI() error {
	debug := os.Getenv("DEBUGNOM")
	if debug != "" {
		f, err := tea.LogToFile(debug, "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	its, err := c.GetAllFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	var errorItems []ErrorItem
	// if no feeds in store, fetchAllFeeds, which will return previews
	if len(c.config.PreviewFeeds) > 0 {
		its, errorItems, err = c.fetchAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}
		// if no items, fetchAllFeeds and GetAllFeeds
	} else if len(its) == 0 {
		_, errorItems, err = c.fetchAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}
		// refetch for consistent data across calls
		its, err = c.GetAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}
	}

	items := convertItems(its)

	es := []string{}
	for _, e := range errorItems {
		es = append(es, fmt.Sprintf("Error fetching %s: %s", e.FeedURL, e.Err))
	}

	if err := Render(items, c, es, c.config); err != nil {
		return fmt.Errorf("commands.TUI: %w", err)
	}

	return nil
}

func (c Commands) List(numResults int) error {
	its, err := c.GetAllFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	output := ""

	for _, item := range its {
		output += fmt.Sprintf("%s \n  - %s\n", item.Title, item.Link)
	}

	if c.config.Pager == "false" {
		fmt.Println(output)
		return nil
	}

	return outputToPager(output)
}

func (c Commands) Add(url string) error {
	err := c.config.AddFeed(config.Feed{URL: url})
	if err != nil {
		return fmt.Errorf("commands Add: %w", err)
	}

	return nil
}

type FetchResultError struct {
	res rss.RSS
	err error
	url string
}

type ErrorItem struct {
	FeedURL string
	Err     error
}

func (c Commands) fetchAllFeeds() ([]store.Item, []ErrorItem, error) {
	var (
		items      []store.Item
		wg         sync.WaitGroup
		errorItems []ErrorItem
	)

	feeds := c.config.GetFeeds()

	if len(feeds) <= 0 {
		return items, errorItems, fmt.Errorf("no feeds found, add to nom/config.yml")
	}

	ch := make(chan FetchResultError)

	for _, feed := range feeds {
		wg.Add(1)

		go fetchFeed(ch, &wg, feed, c.config.Version)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		if result.err != nil {
			errorItems = append(errorItems, ErrorItem{FeedURL: result.url, Err: result.err})
			continue
		}

		for _, r := range result.res.Channel.Items {
			i := store.Item{
				Author:      r.Author,
				Content:     r.Content,
				FeedURL:     result.url,
				FeedName:    r.FeedName,
				Link:        r.Link,
				PublishedAt: r.PubDate,
				Title:       r.Title,
			}

			// only store if non-preview feed
			if !includes(c.config.PreviewFeeds, config.Feed{URL: result.url}) {
				err := c.store.UpsertItem(i)
				if err != nil {
					log.Fatalf("[commands.go] fetchAllFeeds: %e", err)
					continue
				}
			}

			items = append(items, i)
		}
	}

	return items, errorItems, nil
}

func includes[T comparable](arr []T, item T) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

func (c Commands) GetAllFeeds() ([]store.Item, error) {
	err := c.CleanFeeds()
	if err != nil {
		return []store.Item{}, fmt.Errorf("[commands.go] GetAllFeeds: %w", err)
	}

	is, err := c.store.GetAllItems()
	if err != nil {
		return []store.Item{}, fmt.Errorf("commands.go: GetAllFeeds %w", err)
	}

	if c.config.ShowFavourites {
		is = onlyFavourites(is)
	} else if c.config.ShowRead {
		is = showRead(is)
	} else {
		is = defaultView(is)
	}

	// add FeedName from config for custom names
	for i := 0; i < len(is); i++ {
		for _, f := range c.config.Feeds {
			if f.URL == is[i].FeedURL {
				is[i].FeedName = f.Name
			}
		}
	}

	return is, nil
}

func onlyFavourites(items []store.Item) (is []store.Item) {
	for _, v := range items {
		if v.Favourite {
			is = append(is, v)
		}
	}

	return is
}

// currently showRead shows all items
func showRead(items []store.Item) (is []store.Item) {
	return items
}

func defaultView(items []store.Item) (is []store.Item) {
	for _, v := range items {
		if !v.Read() {
			is = append(is, v)
		}
	}

	return is
}

func fetchFeed(ch chan FetchResultError, wg *sync.WaitGroup, feed config.Feed, version string) {
	defer wg.Done()

	r, err := rss.Fetch(feed, version)

	if err != nil {
		ch <- FetchResultError{res: rss.RSS{}, err: err, url: feed.URL}
		return
	}

	ch <- FetchResultError{res: r, err: nil, url: feed.URL}
}

func (c Commands) GetGlamourisedArticle(ID int) (string, error) {
	article, err := c.store.GetItemByID(ID)
	if err != nil {
		return "", fmt.Errorf("commands.FindGlamourisedArticle: %w", err)
	}

	if c.config.AutoRead {
		err = c.store.ToggleRead(article.ID)
		if err != nil {
			return "", fmt.Errorf("[commands.go] GetGlamourisedArticle: %w", err)
		}
	}

	content, err := glamouriseItem(article, c.config.Theme)
	if err != nil {
		return "", fmt.Errorf("[commands.go] GetGlamourisedArticle: %w", err)
	}

	return content, nil
}

func getStyleConfigWithOverrides(theme config.Theme) (sc ansi.StyleConfig) {
	switch theme.Glamour {
	case "light":
		sc = glamour.LightStyleConfig
	case "dracula":
		sc = glamour.DraculaStyleConfig
	case "pink":
		sc = glamour.PinkStyleConfig
	case "ascii":
		sc = glamour.ASCIIStyleConfig
	case "notty":
		sc = glamour.NoTTYStyleConfig
	default:
		sc = glamour.DarkStyleConfig
	}

	sc.H1.BackgroundColor = &theme.TitleColor

	return sc
}

func glamouriseItem(item store.Item, theme config.Theme) (string, error) {
	var mdown string

	mdown += "# " + item.Title
	mdown += "\n"
	mdown += item.Author
	if !item.PublishedAt.IsZero() {
		mdown += "\n"
		mdown += item.PublishedAt.String()
	}
	mdown += "\n\n"
	mdown += item.Link
	mdown += "\n\n"
	mdown += htmlToMd(item.Content)

	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(getStyleConfigWithOverrides(theme)),
	)

	out, err := r.Render(mdown)
	if err != nil {
		return "", fmt.Errorf("GlamouriseItem: %w", err)
	}

	return out, nil
}

func htmlToMd(html string) string {
	converter := md.NewConverter("", true, nil)

	mdown, err := converter.ConvertString(html)
	if err != nil {
		log.Fatal(err)
	}

	return mdown
}

func outputToPager(content string) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -r"
	}

	pa := strings.Split(pager, " ")
	cmd := exec.Command(pa[0], pa[1:]...)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout

	return cmd.Run()
}
