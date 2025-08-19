package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

func WithVPrefix(v string) string {
	// v: "go1.25.0" or "1.25.0"
	v = strings.TrimPrefix(v, "go")
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

func cmp(a, b string) int {
	return semver.Compare(WithVPrefix(a), WithVPrefix(b))
}

func main() {
	gid := os.Getgid()
	uid := os.Getuid()

	if gid != 0 || uid != 0 {
		log.Fatalln("You need to run this as root")
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

		switch x := cmp(rv, local); {
		case x > 0:
			rmt = append(rmt, remote{idx: i, version: rv})
		case x == 0:
			fmt.Println("no updates available")
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

			extractCmd := []string{"tar", "-xzvf", dlPath}
			execCmd(extractCmd)

			src := "/tmp/go"

			err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				err = os.Chown(path, 0, 0)
				if err != nil {
					return err
				}

				dest := "/usr/local/go"
				if _, err := os.Stat(dest); err == nil {
					if err := os.RemoveAll(dest); err != nil {
						return err
					}
				}
				if err := os.Rename(src, dest); err != nil {
					return err
				}
				return nil

			})
			if err != nil {
				log.Fatal(err)
			}

		}
	}
}

func execCmd(args []string) {
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func Limit[T any](xs []T, k int) []T {
	if k <= 0 || len(xs) <= k {
		return xs
	}
	return xs[:k]
}

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
