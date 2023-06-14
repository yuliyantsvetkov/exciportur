package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	commitAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_last_commit_age_seconds",
			Help: "Age of the last commit in seconds",
		},
		[]string{"repository", "commit_sha"}, // Include "commit_sha" as a label
	)

	scrapeInterval time.Duration
	accessToken    string
	repositories   []string
)

func init() {
	prometheus.MustRegister(commitAge)
}

func main() {
	// Read environment variables
	repoNames := os.Getenv("REPO_NAMES")
	scrapeIntervalStr := os.Getenv("SCRAPE_INTERVAL")
	accessToken = os.Getenv("ACCESS_TOKEN")
	logLevel := os.Getenv("LOG_LEVEL")

	// Parse scrape interval
	var err error
	scrapeInterval, err = time.ParseDuration(scrapeIntervalStr)
	if err != nil {
		log.Fatalf("Failed to parse SCRAPE_INTERVAL: %v", err)
	}

	// Parse repositories
	repositories = strings.Split(repoNames, ",")

	// Set up logging
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.SetOutput(os.Stdout)
	if logLevel == "DEBUG" {
		logger.SetLevel(logrus.DebugLevel)
	}

	// Start the exporter
	go updateCommitMetrics(logger)

	http.Handle("/metrics", promhttp.Handler())

	logger.Println("Starting Prometheus exporter on :8000")
	logger.Fatal(http.ListenAndServe(":8000", nil))
}

func updateCommitMetrics(logger *logrus.Logger) {
	for {
		for _, repo := range repositories {
			repoParts := strings.SplitN(repo, "/", 2)
			if len(repoParts) != 2 {
				logger.Printf("Invalid repository: %s", repo)
				continue
			}
			repoOwner, repoName := repoParts[0], repoParts[1]

			lastCommitTime, commitSHAValue, err := fetchRecentCommitInfo(repoOwner, repoName)
			if err != nil {
				logger.Printf("Failed to fetch recent commit info for repository %s: %v", repo, err)
				continue
			}

			commitAge.With(prometheus.Labels{"repository": repo, "commit_sha": commitSHAValue[:8]}).Set(time.Since(lastCommitTime).Seconds())

			logger.WithFields(logrus.Fields{
				"repository": repo,
				"age":        time.Since(lastCommitTime).Seconds(),
				"sha":        commitSHAValue[:8], // Extract the first 8 characters
			}).Debug("Metrics")
		}

		time.Sleep(scrapeInterval)
	}
}

func fetchRecentCommitInfo(repoOwner, repoName string) (time.Time, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", repoOwner, repoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "token "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("failed to fetch recent commit info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var commits []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			return time.Time{}, "", fmt.Errorf("failed to decode response body: %v", err)
		}

		if len(commits) > 0 {
			commit := commits[0]
			commitDate, err := time.Parse(time.RFC3339, commit["commit"].(map[string]interface{})["committer"].(map[string]interface{})["date"].(string))
			if err != nil {
				return time.Time{}, "", fmt.Errorf("failed to parse commit date: %v", err)
			}
			commitSHAValue := commit["sha"].(string)
			return commitDate, commitSHAValue, nil
		}
	}

	return time.Time{}, "", fmt.Errorf("no commits found")
}
