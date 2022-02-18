package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	GRAFANA_SEARCH_PATH        = "api/search/"
	GRAFANA_GET_DASHBOARD_PATH = "api/dashboards/uid"
)

type SearchResult struct {
	Uid   string
	Title string
}

type DashboardMeta struct {
	Slug        string
	Provisioned bool
}

type Dashboard struct {
	Meta      DashboardMeta
	Dashboard interface{}
}

type DashboardInfo struct {
	SearchResult
	DashboardJson []byte
}

func main() {
	grafanaDashboards := getGrafanaDashboard()

	tempDir, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatalln("Create temp directory error: ", err.Error())
	}

	gitAuth := gitHttp.BasicAuth{
		Username: os.Getenv("GIT_USER"),
		Password: os.Getenv("GIT_PASSWD"),
	}
	gitRepo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:  os.Getenv("GIT_REPO_URL"),
		Auth: &gitAuth,
	})
	if err != nil {
		log.Fatalln("Can't checkout Git repo: ", err.Error())
	}
	gitAuthor := os.Getenv("GIT_AUTHOR")
	gitAuthorEmail := os.Getenv("GIT_AUTHOR_EMAIL")
	if gitAuthor == "" {
		gitAuthor = "NO BODY"
	}
	if gitAuthorEmail == "" {
		gitAuthorEmail = "no-body@example.com"
	}

	workingTree, _ := gitRepo.Worktree()

	directoryPrefix := os.Getenv("DIR_PREFIX")

	for _, dashboard := range grafanaDashboards {
		targetFile := filepath.Join(directoryPrefix, dashboard.Uid, fmt.Sprintf("%s.json", dashboard.Title))
		targetFullPath := filepath.Join(tempDir, targetFile)

		// Make sure parent directory exist
		os.MkdirAll(filepath.Dir(targetFullPath), 0777)
		ioutil.WriteFile(targetFullPath, dashboard.DashboardJson, 0644)
		workingTree.Add(targetFile)
	}

	gitStatus, _ := workingTree.Status()
	if len(gitStatus) > 0 {
		_, err = workingTree.Commit(fmt.Sprintf("%d dashboard(s) content updated.", len(gitStatus)), &git.CommitOptions{
			Author: &object.Signature{
				Name:  gitAuthor,
				Email: gitAuthorEmail,
				When:  time.Now(),
			},
		})
		if err != nil {
			log.Fatalln("Commit failed: ", err.Error())
		}
		err = gitRepo.Push(&git.PushOptions{Auth: &gitAuth})
		if err != nil {
			log.Fatalln("Push to git remote failed: ", err.Error())
		}
		log.Println("Change pushed to repo.")
	} else {
		log.Println("Nothing changed.")
	}
}

func getGrafanaDashboard() []DashboardInfo {
	client := &http.Client{}
	grafanaUrl := os.Getenv("GRAFANA_URL")
	grafanaTokenValue := os.Getenv("GRAFANA_TOKEN")
	grafanaToken := fmt.Sprintf("Bearer %s", grafanaTokenValue)

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", grafanaUrl, GRAFANA_SEARCH_PATH), nil)
	req.Header.Set("Authorization", grafanaToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	q := req.URL.Query()
	q.Add("type", "dash-db")
	req.URL.RawQuery = q.Encode()
	res, _ := client.Do(req)

	defer res.Body.Close()
	resp_body, _ := ioutil.ReadAll(res.Body)

	var searchResults []SearchResult
	var results []DashboardInfo

	json.Unmarshal(resp_body, &searchResults)

	for _, searchResult := range searchResults {
		dashboardRequest, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", grafanaUrl, GRAFANA_GET_DASHBOARD_PATH, searchResult.Uid), nil)
		dashboardRequest.Header.Set("Authorization", grafanaToken)
		dashboardRequest.Header.Set("Accept", "application/json")
		dashboardRequest.Header.Set("Content-Type", "application/json")
		dashboardResponse, _ := client.Do(dashboardRequest)
		defer dashboardResponse.Body.Close()
		dashboardBody, _ := ioutil.ReadAll(dashboardResponse.Body)

		var dashboard Dashboard
		var dashboardInfo DashboardInfo

		json.Unmarshal(dashboardBody, &dashboard)
		if !dashboard.Meta.Provisioned {
			dashboardInfo.Uid = searchResult.Uid
			dashboardInfo.Title = searchResult.Title
			dashboardInfo.DashboardJson, _ = json.Marshal(dashboard.Dashboard)
			results = append(results, dashboardInfo)
		}
	}
	return results
}
