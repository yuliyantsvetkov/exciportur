# exciportur

# Brief introduction
Prometheus exporter to get the age of the last commit and the short SHA.

The idea of this exporter is to bring to Prometheus metrics which can show the cadance of commits to repositories for GitOps oriented companies.

It exports the SHORT SHA of the last commit and the age of the commit.

# How to run
Its simple `Golang` application. You can run it will following commands:
```
go mod init
go mod tidy
go build
```

# Configuration

`REPO_NAMES` - comma separated list of repositories which to monitor. For example: `org/repository,org/repository2`.

`SCRAPE_INTERVAL` - Github API scrape interval in seconds.

`ACCESS_TOKEN` - Github Personal Access token.

`LOG_LEVEL` - Log level configuration variable for the JSON logging. For example: `INFO|WARN|DEBUG`.