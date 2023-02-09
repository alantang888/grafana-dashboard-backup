package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
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
	GRAFANA_ALERT_RULE_PATH    = "api/ruler/grafana/api/v1/rules"
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
	dashboardExportMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "grafana_export_dashboard_export_total",
		Help: "Number of grafana dashboard exported",
	})
	alertruleExportMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "grafana_export_alert_rule_export_total",
		Help: "Number of grafana alert rule exported",
	})
	gitWorktreeStatusMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "grafana_export_git_status_total",
		Help: "Nuber of item changed on git",
	})

	grafanaDashboards := getGrafanaDashboard()
	alertRuleNamespaces := getGrafanaAlertRule()

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

		dashboardExportMetric.Inc()
	}

	alertRuleDir := os.Getenv("ALERT_RULE_DIR_PREFIX")
	for arNamespace, arGroups := range alertRuleNamespaces {
		for _, arGroup := range arGroups {
			targetFile := filepath.Join(alertRuleDir, arNamespace, fmt.Sprintf("%s.json", arGroup.Name))
			targetFullPath := filepath.Join(tempDir, targetFile)

			os.MkdirAll(filepath.Dir(targetFullPath), 0777)
			arGroupJson, err := json.MarshalIndent(arGroup, "", "  ")
			if err != nil {
				log.Panic(err)
			}
			ioutil.WriteFile(targetFullPath, arGroupJson, 0644)
			workingTree.Add(targetFile)

			alertruleExportMetric.Inc()
		}
	}

	gitStatus, _ := workingTree.Status()
	if len(gitStatus) > 0 {
		_, err = workingTree.Commit(fmt.Sprintf("%d file(s) content updated.", len(gitStatus)), &git.CommitOptions{
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
	gitWorktreeStatusMetric.Set(float64(len(gitStatus)))

	pushUrl := os.Getenv("PUSH_GATEWAY_URL")
	pushJobName := os.Getenv("PUSH_JOB_NAME")
	if pushUrl != "" {
		if pushJobName == "" {
			pushJobName = "grafana_export"
		}
		push.New(pushUrl, pushJobName).Collector(dashboardExportMetric).Collector(alertruleExportMetric).Collector(gitWorktreeStatusMetric).Push()
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
			dashboardInfo.DashboardJson, _ = json.MarshalIndent(dashboard.Dashboard, "", "  ")
			results = append(results, dashboardInfo)
		}
	}
	return results
}

func getGrafanaAlertRule() NamespaceConfigResponse {
	client := &http.Client{}
	grafanaUrl := os.Getenv("GRAFANA_URL")
	grafanaTokenValue := os.Getenv("GRAFANA_TOKEN")
	grafanaToken := fmt.Sprintf("Bearer %s", grafanaTokenValue)

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", grafanaUrl, GRAFANA_ALERT_RULE_PATH), nil)
	req.Header.Set("Authorization", grafanaToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		log.Panic("Can't fetch grafana alert rules. Error: ", err.Error())
	}

	defer res.Body.Close()
	resp_body, _ := ioutil.ReadAll(res.Body)

	var alertRuleNamespaces NamespaceConfigResponse
	err = json.Unmarshal(resp_body, &alertRuleNamespaces)
	if err != nil {
		log.Panic("Can't parse grafana alert rules json. Error: ", err.Error())
	}
	return alertRuleNamespaces
}
