# Bitbucket Exporter

Prometheus exporter for Bitbucket. Collecting bitbucket data at background and exposes it as matric. We fetched the data via <a href="https://developer.atlassian.com/cloud/bitbucket/rest/intro">Cloud Rest</a>.


## Configuration
Configuration defined at yaml file. Now, we only support bitbucket basic auth implementation.

```yaml
---
auth:
  type: "basic"
  basic:
    # your bitbucket email
    username: ""
    # your bitbucket api token/app password
    password: ""
included_workspaces: ["your_workspace_slug"]
refs_collector:
  # collect total branch at repo
  collect_total_branch: true
  # collect total tag at repo
  collect_total_tag: true
commit_collector:
  included_repository: ["your_workspace_slug/your_repo_slug"]
  # count total commit at repo
  collect_total_commit_repo: true
  # count total commit of user at repo
  collect_total_commit_user: true
```

Basic auth bitbucket :
- <a href="https://developer.atlassian.com/cloud/bitbucket/rest/intro#app-passwords">App passwords</a>
- <a href="https://developer.atlassian.com/cloud/bitbucket/rest/intro#api-tokens">Api Token</a>



## Quick Start
### Run Locally

Clone Repo :

```bash
git clone https://github.com/nandanurseptama/bitbucket-exporter.git
cd bitbucket-exporter
```

Setup Config:

```bash
cp example-config.yaml config.yaml
```
Then edit config file to your preferences.


Build locally :
```bash
go build -o bitbucket_exporter cmd/bitbucket_exporter/main.go
```

Test with:

```bash
curl "http://localhost:9171/metrics"
```

### Example Prometheus config:

```yaml
scrape_configs:
  - job_name: postgres
    static_configs:
      - targets: ["127.0.0.1:9171"] # Replace IP with the hostname of the docker container if you're running the container in a separate network
```
