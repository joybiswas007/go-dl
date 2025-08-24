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
	"runtime"
	"strings"
)

const (
	BASE_URL = "https://go.dev/dl/"
)

func main() {
	// Check required dependencies
	cmds := []string{"wget", "tar"}
	for _, cmd := range cmds {
		if _, err := exec.LookPath(cmd); err != nil {
			fmt.Printf("❌ Required command \"%s\" is not installed. Please install it to proceed.\n", cmd)
			return
		}
	}

	values := url.Values{}
	values.Add("mode", "json")
	urlWithParams := fmt.Sprintf("%s?%s", BASE_URL, values.Encode())

	releases, err := GetReleases(&http.Client{}, urlWithParams)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Available versions...")
	for i, r := range releases {
		if r.Stable {
			fmt.Printf("%d. %s (stable)\n", i+1, withVPrefix(r.Version))
		} else {
			fmt.Printf("%d. %s (unstable)\n", i+1, withVPrefix(r.Version))
		}
	}
	var input int
	fmt.Printf("Enter your choice (1-%d): ", len(releases))
	fmt.Scanln(&input)

	if input == 0 || input > len(releases) {
		log.Fatalf("Please select a number between 1 and %d\n", len(releases))
	}

	release := releases[input-1]
	fmt.Printf("Selected: %s\n\n", withVPrefix(release.Version))

	fmt.Println("Starting installation...")
	downloadAndInstallGo(release)
}

// withVPrefix normalizes a Go version string to have a leading "v"
func withVPrefix(v string) string {
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

func downloadAndInstallGo(release Release) {
	for _, file := range release.Files {
		if runtime.GOARCH == file.Arch && runtime.GOOS == file.Os {
			dlURL := fmt.Sprintf("%s%s", BASE_URL, file.Filename)
			dlCmd := []string{"wget", "-c", "--tries=5", "--read-timeout=10", "-P", "/tmp", dlURL}
			execCmd(dlCmd)

			dlPath := fmt.Sprintf("/tmp/%s", file.Filename)

			// unarchive
			execCmd([]string{"tar", "-xzvf", dlPath, "-C", "/tmp"})

			// delete the archive file
			execCmd([]string{"rm", "-rf", dlPath})

			// Now check if already installed version exist
			// If exist remove the directory
			existingDir := "/usr/local/go"
			if _, err := os.Stat(existingDir); err == nil {
				execCmd([]string{"sudo", "rm", "-rf", existingDir})
			}

			src := "/tmp/go"

			// change permission
			execCmd([]string{"sudo", "chown", "-R", "root:root", src})

			// move to /usr/local
			execCmd([]string{"sudo", "mv", "-v", src, "/usr/local"})

			execCmd([]string{"go", "version"})
		}
	}
}

// GetReleases fetches release metadata from baseURL using the provided HTTP client.
func GetReleases(client *http.Client, baseURL string) (Releases, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("referer", BASE_URL)
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

type Releases []Release

type Release struct {
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
