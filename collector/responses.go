package collector

// Response wrapper for pagination bitbucket
type PaginationResponse[T any] struct {
	PageLen uint64  `json:"pagelen"`
	Next    *string `json:"next"`
	Values  []T     `json:"values"`
}

type Repositories struct {
	Slug      string    `json:"slug"`
	Uuid      string    `json:"uuid"`
	Name      string    `json:"name"`
	FullName  string    `json:"full_name"`
	Language  string    `json:"language"`
	Workspace Workspace `json:"workspace"`
	Project   Project   `json:"project"`
}

type Workspace struct {
	Slug string `json:"slug"`
	Uuid string `json:"uuid"`
	Name string `json:"name"`
}

type Project struct {
	Key  string `json:"key"`
	Uuid string `json:"uuid"`
	Name string `json:"name"`
}
