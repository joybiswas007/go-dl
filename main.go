// Package main
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
)

const (
	BaseURL = "https://go.dev/dl/"
)

var banner = `
 ██████╗  ██████╗       ██████╗ ██╗     
██╔════╝ ██╔═══██╗      ██╔══██╗██║     
██║  ███╗██║   ██║█████╗██║  ██║██║     
██║   ██║██║   ██║╚════╝██║  ██║██║     
╚██████╔╝╚██████╔╝      ██████╔╝███████╗
 ╚═════╝  ╚═════╝       ╚═════╝ ╚══════╝
`

func main() {
	fmt.Println(banner)
	checkGOPATH()
	var doctor bool

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "go-dl - Automate Go download & install setup\n")
		fmt.Fprintf(os.Stderr, "=========================\n\n")

		fmt.Fprintf(os.Stderr, " Why go-dl?\n")
		fmt.Fprintf(os.Stderr, "   • Just select a version and let go-dl handle everything!\n\n")
	}
	flag.BoolVar(&doctor, "doctor", false, "Run a system check to verify that all required packages are installed")
	flag.Parse()

	if doctor {
		cmds := []string{"wget", "tar", "sudo", "chown"}
		for _, cmd := range cmds {
			if _, err := exec.LookPath(cmd); err != nil {
				fmt.Printf("❌ \"%s\" is not installed. Please install it to proceed.\n", cmd)
				continue
			}
		}
		fmt.Println("Dependencies check passed successfully")
		return
	}

	values := url.Values{}
	values.Add("mode", "json")
	urlWithParams := fmt.Sprintf("%s?%s", BaseURL, values.Encode())

	releases, err := getReleases(&http.Client{}, urlWithParams)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Architecture: %s\n\n", runtime.GOARCH)
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

// withVPrefix normalizes a Go version string to have a leading "v".
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
		// If Arch doesn't match OR OS doesn't match -> Skip (continue)
		if runtime.GOARCH != file.Arch || runtime.GOOS != file.Os {
			continue
		}
		dlURL := fmt.Sprintf("%s%s", BaseURL, file.Filename)
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

func checkGOPATH() {
	if _, isSet := os.LookupEnv("GOPATH"); !isSet {
		fmt.Println("$GOPATH environment variable is not set.")
		fmt.Println("Please add the following lines to your ~/.bashrc or ~/.zshrc file, then restart your shell:")
		fmt.Println()
		fmt.Println("export GOPATH=$HOME/go")
		fmt.Println("export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin")
		fmt.Println()
	}
}

// getReleases fetches release metadata from baseURL using the provided HTTP client.
func getReleases(client *http.Client, dlURL string) (Releases, error) {
	req, err := http.NewRequest(http.MethodGet, dlURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("referer", BaseURL)
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
