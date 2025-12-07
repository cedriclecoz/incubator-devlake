/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

type GithubCheckRun struct {
	ConnectionId    uint64 `gorm:"primaryKey"`
	GithubId        int64  `gorm:"primaryKey"`              // GitHub's unique check run ID
	RepoId          uint64 `gorm:"index"`                   // Repository ID for filtering
	HeadSha         string `gorm:"index;type:varchar(40)"` // Index for querying by commit
	ExternalId      string `gorm:"type:varchar(255)"`      // External ID (e.g., from CI system)
	Name            string `gorm:"index;type:varchar(255)"` // Name of the check (e.g., "build", "test")
	Status          string `gorm:"type:varchar(100)"`      // queued, in_progress, completed
	Conclusion      string `gorm:"type:varchar(100)"`      // success, failure, neutral, cancelled, skipped, timed_out, action_required
	DetailsUrl      string `gorm:"type:varchar(255)"`      // URL to see more details
	HtmlUrl         string `gorm:"type:varchar(255)"`      // HTML URL
	CheckSuiteId    int64  `gorm:"index"`                  // Associated check suite ID
	AppId           int64                                  // GitHub App ID that created this
	AppName         string `gorm:"type:varchar(255)"`      // Name of the GitHub App
	AppSlug         string `gorm:"type:varchar(255)"`      // Slug of the GitHub App
	GithubStartedAt *time.Time
	GithubCompletedAt *time.Time
	common.NoPKModel
}

func (GithubCheckRun) TableName() string {
	return "_tool_github_check_runs"
}
