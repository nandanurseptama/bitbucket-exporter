package collector

const (
	namespace = "bitbucket"
)

// subsystem name
const (
	subSystemRepositories = "repositories"
	subSystemMember       = "member"
	subSystemRepoRefs     = "repository_refs"
	subSystemCommit       = "commit"
)

// key for mapping collectors
const (
	keyScrapeCollector       = "scrape"
	keyRepositoriesCollector = "repositories"
	keyMemberCollector       = "member"
	keyRefsCollector         = "refs"
	keyCommitCollector       = "commit"
)

// endpoint
const (
	repositoriesEndpoint         = "repositories"
	workspaceMembersEndpoint     = "workspaces/:workspace/members"
	refsRepositoryEndpoint       = "repositories/:workspace/:repo_slug/refs"
	listCommitRepositoryEndpoint = "repositories/:workspace_repo_slug/commits"
)
