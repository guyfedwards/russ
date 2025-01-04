package commands

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func updateViewport(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.ResumeMsg:
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ViewportKeyMap.Suspend):
			return m, tea.Suspend
		case key.Matches(msg, ViewportKeyMap.GotoStart):
			m.viewport.GotoTop()

		case key.Matches(msg, ViewportKeyMap.GotoEnd):
			m.viewport.GotoBottom()

		case key.Matches(msg, ViewportKeyMap.Escape):
			// if last nav was marked read navigate to last read not new
			if m.markedRead && !m.commands.config.ShowRead && m.commands.config.ShowLastRead {
				current := m.list.Index()
				if current-1 >= 0 {
					m.list.Select(current - 1)
				}
			}

			m.selectedArticle = nil
			m.markedRead = false

		case key.Matches(msg, ViewportKeyMap.OpenInBrowser):
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				return m, nil
			}

			it := ItemToTUIItem(current)
			cmd = m.commands.OpenLink(it.URL)
			cmds = append(cmds, cmd)

		case key.Matches(msg, ViewportKeyMap.Favourite):
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				return m, nil
			}
			err = m.commands.store.ToggleFavourite(current.ID)
			if err != nil {
				return m, tea.Quit
			}
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.Read):
			if m.commands.config.AutoRead {
				return m, nil
			}
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				return m, nil
			}
			err = m.commands.store.ToggleRead(current.ID)
			if err != nil {
				return m, tea.Quit
			}

			// don't use special nav when showing read posts
			if !m.commands.config.ShowRead {
				m.markedRead = true
			}
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.Prev):
			current := m.list.Index()
			if current-1 < 0 {
				return m, nil
			}

			navIndex := getPrevIndex(current)
			m.list.Select(navIndex)
			items := m.list.Items()
			item := items[navIndex]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				return m, tea.Quit
			}

			m.markedRead = false
			m.viewport.SetContent(content)

		case key.Matches(msg, ViewportKeyMap.Next):
			current := m.list.Index()
			items := m.list.Items()
			if current+1 >= len(items) {
				return m, nil
			}

			navIndex := getNextIndex(m.markedRead, current)
			m.list.Select(navIndex)
			item := items[navIndex]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				return m, tea.Quit
			}

			m.markedRead = false
			m.viewport.SetContent(content)
		case key.Matches(msg, ViewportKeyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, ViewportKeyMap.ShowFullHelp):
			m.help.ShowAll = !m.help.ShowAll
			if m.help.ShowAll {
				m.viewport.Height = m.viewport.Height + lipgloss.Height(m.help.ShortHelpView(ViewportKeyMap.ShortHelp())) - lipgloss.Height(m.help.FullHelpView(ViewportKeyMap.FullHelp()))
			} else {
				m.viewport.Height = m.viewport.Height + lipgloss.Height(m.help.FullHelpView(ViewportKeyMap.FullHelp())) - lipgloss.Height(m.help.ShortHelpView(ViewportKeyMap.ShortHelp()))
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func getPrevIndex(current int) int {
	return current - 1
}

func getNextIndex(marked bool, current int) int {
	if marked {
		return current
	}
	return current + 1
}

func viewportView(m model) string {
	return m.viewport.View() + "\n" + m.viewportHelp()
}

func (m model) viewportHelp() string {
	return helpStyle.Render(m.help.View(ViewportKeyMap))
}
