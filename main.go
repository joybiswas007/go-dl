package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

func main() {
	var (
		doctor bool
		check  bool
	)

	flag.BoolVar(&doctor, "doctor", false, "Run a system check to verify that all required packages are installed")
	flag.BoolVar(&check, "check", false, "Check if a new version of Go is available")
	flag.Parse()

	if doctor {
		cmds := []string{"wget", "tar"}
		for _, cmd := range cmds {
			if _, err := exec.LookPath(cmd); err != nil {
				fmt.Printf("❌ \"%s\" is not installed. Please install it to proceed.\n", cmd)
				continue
			}
		}
		fmt.Println("Dependencies check passed successfully.")
		return
	}

	baseURL := "https://go.dev/dl/"

	values := url.Values{}
	values.Add("mode", "json")
	urlWithParams := fmt.Sprintf("%s?", baseURL) + values.Encode()

	releases, err := GetReleases(&http.Client{}, urlWithParams)
	if err != nil {
		log.Fatal(err)
	}

	local := WithVPrefix(runtime.Version())
	var rmt []remote
	for i, r := range releases {
		rv := WithVPrefix(r.Version)

		// If both version are same continue to next version
		// And we compare them together
		if local == rv {
			continue
		}

		switch semver.Compare(local, rv) {
		case -1:
			if check {
				fmt.Printf("New version available! Current: %s, Latest: %s\n", local, rv)
				return
			}
			rmt = append(rmt, remote{idx: i, version: rv})
		case 0, 1:
			fmt.Println("You are using the latest version.")
			return
		}
	}
	new := releases[Limit(rmt, 1)[0].idx]
	var dlURL string
	for _, f := range new.Files {
		if runtime.GOARCH == f.Arch && runtime.GOOS == f.Os {
			dlURL = fmt.Sprintf("%s%s", baseURL, f.Filename)
			dlCmd := []string{"wget", "-c", "--tries=5", "--read-timeout=10", "-P", "/tmp", dlURL}
			execCmd(dlCmd)

			dlPath := fmt.Sprintf("/tmp/%s", f.Filename)

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
