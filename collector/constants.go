package collector

const (
	namespace = "bitbucket"
)

// subsystem name
const (
	subSystemRepositories = "repositories"
	subSystemMember       = "member"
)

// key for mapping collectors
const (
	keyScrapeCollector       = "scrape"
	keyRepositoriesCollector = "repositories"
	keyMemberCollector       = "member"
)

// endpoint
const (
	repositoriesEndpoint     = "repositories"
	workspaceMembersEndpoint = "workspaces/:workspace/members"
)
