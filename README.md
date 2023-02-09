# Grafana Dashboard Backup to Git
Export grafana dashboard, alert rule then commit to Git repo.

Since dashboard title can change. And that sill the same dashboard. So the file structure use `PERFIX/UID/TITLE.json`.

Alert rule structure use `ALERTRULE_NAMESPACE/ALERTRULE_GROUP.json`. 
If multiple rule using same alert group. They will store in same file.

Environment variables:
- `GIT_REPO_URL`: Git repo. Current only using HTTP. Not SSH (Just lazy to set SSH keys...)
- `GIT_USER`: Git username for basic auth
- `GIT_PASSWD`: Git password for basic auth
- `GIT_AUTHOR`: Git author (default: "NO BODY")
- `GIT_AUTHOR_EMAIL`: Git author email (default: "no-body@example.com")
- `DIR_PREFIX`: Dashboard directory prefix on git repo
- `ALERT_RULE_DIR_PREFIX`: Alert rule directory prefix on git repo
- `GRAFANA_URL`: Grafana URL
- `GRAFANA_TOKEN`: Grafana API token. It just need viewer permission
