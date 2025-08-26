# go-dl

**Automate Go Setup**

**Note**: _`go-dl` IS NOT A VERSION MANAGER. `go-dl` simply eliminates the tedious process of manually downloading Go from Go's website, extracting the tar file, setting up permissions, and moving to /usr/local._


## Prerequisites

Make sure the following tools are installed on your system:
- `go` 
- `wget` - for downloading Go releases
- `tar` - for extracting archives

## Installation
### Method 1 (Recommended)
```
go install github.com/joybiswas007/go-dl@latest   
```
### Method 2
### Clone and Build
```
git clone https://github.com/joybiswas007/go-dl
cd go-dl
go mod tidy
go build -o go-dl main.go
sudo mv go-dl /usr/local/bin/
```

## Usage

```
go-dl

# Show help
go-dl --help
```
