package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const BASE_URL = "https://go.dev/dl/"

var (
	releases Releases
)

func main() {
	cmds := []string{"wget", "tar"}
	for _, cmd := range cmds {
		if _, err := exec.LookPath(cmd); err != nil {
			fmt.Printf("❌ \"%s\" is not installed. Please install it to proceed.\n", cmd)
			return
		}
	}

	values := url.Values{}
	values.Add("mode", "json")
	urlWithParams := fmt.Sprintf("%s?", BASE_URL) + values.Encode()

	rls, err := GetReleases(&http.Client{}, urlWithParams)
	if err != nil {
		log.Fatal(err)
	}
	releases = rls

	var items []list.Item
	for _, r := range releases {
		items = append(items, item(r.Version))
	}

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Versions available for download"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}

// WithVPrefix normalizes a Go version string to have a leading "v" as required by semver.Compare.
func WithVPrefix(v string) string {
	v = strings.TrimPrefix(v, "go")
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// execCmd runs an external command and streams its stdin/stdout/stderr to the current process.
func execCmd(args []string) {
	cmd := exec.Command(args[0], args[1:]...)

	// Pipe the command's stdin/stderr/stdout to this process's stdin/stderr/stdout.
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

// Limit returns at most the first k elements of xs.
// If k <= 0 or xs has length <= k, it returns xs unchanged.
func Limit[T any](xs []T, k int) []T {
	if k <= 0 || len(xs) <= k {
		return xs
	}
	return xs[:k]
}

// GetReleases fetches release metadata from baseURL using the provided HTTP client.
func GetReleases(client *http.Client, baseURL string) (Releases, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("referer", "https://go.dev/dl/")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	var rl Releases
	if err := json.NewDecoder(resp.Body).Decode(&rl); err != nil {
		return nil, err
	}
	return rl, nil
}

type remote struct {
	idx     int
	version string
}

type Releases []struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []File `json:"files"`
}

type File struct {
	Filename string `json:"filename"`
	Os       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Sha256   string `json:"sha256"`
	Size     int    `json:"size"`
	Kind     string `json:"kind"`
}
