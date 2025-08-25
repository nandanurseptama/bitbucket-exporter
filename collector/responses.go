// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import "time"

// Response wrapper for pagination bitbucket
type PaginationResponse[T any] struct {
	PageLen uint64  `json:"pagelen"`
	Next    *string `json:"next"`
	Values  []T     `json:"values"`
	Size    uint64  `json:"size"`
}

// Response wrapper for repository
type Repository struct {
	Slug      string    `json:"slug"`
	Uuid      string    `json:"uuid"`
	Name      string    `json:"name"`
	FullName  string    `json:"full_name"`
	Language  string    `json:"language"`
	Workspace Workspace `json:"workspace"`
	Project   Project   `json:"project"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Size      uint64    `json:"size"`
	HasIssues bool      `json:"has_issues"`
	HasWiki   bool      `json:"has_wiki"`
	IsPrivate bool      `json:"is_private"`
}

// Response wrapper for workspace
type Workspace struct {
	Slug string `json:"slug"`
	Uuid string `json:"uuid"`
	Name string `json:"name"`
}

// Response wrapper for project
type Project struct {
	Key  string `json:"key"`
	Uuid string `json:"uuid"`
	Name string `json:"name"`
}

type Refs struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Author struct {
	User User `json:"user"`
}

type Commit struct {
	Author Author `json:"author"`
}

type User struct {
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	Uuid        string `json:"uuid"`
}
