package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		for _, release := range releases {
			if m.choice == release.Version {
				for _, file := range release.Files {
					if runtime.GOARCH == file.Arch && runtime.GOOS == file.Os {
						dlURL := fmt.Sprintf("%s%s", BASE_URL, file.Filename)
						dlCmd := []string{"wget", "-c", "--tries=5", "--read-timeout=10", "-P", "/tmp", dlURL}
						execCmd(dlCmd)

						dlPath := fmt.Sprintf("/tmp/%s", file.Filename)

						extractCmd := []string{"tar", "-xzvf", dlPath, "-C", "/tmp"}
						execCmd(extractCmd)

						rmCmd := []string{"rm", "-rf", dlPath}
						execCmd(rmCmd)

						// Now check if already installed version exist
						// If exist remove the directory
						existingDir := "/usr/local/go"
						if _, err := os.Stat(existingDir); err == nil {
							execCmd([]string{"sudo", "rm", "-rf", existingDir})
						}

						src := "/tmp/go"

						permsCmd := []string{"sudo", "chown", "-R", "root:root", src}
						execCmd(permsCmd)

						mvCmd := []string{"sudo", "mv", "-v", src, "/usr/local"}
						execCmd(mvCmd)

						checkGoCmd := []string{"go", "version"}
						execCmd(checkGoCmd)
					}
				}

			}
		}
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return quitTextStyle.Render("Not hungry? That’s cool.")
	}
	return "\n" + m.list.View()
}
