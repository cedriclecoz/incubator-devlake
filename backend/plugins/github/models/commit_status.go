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

type GithubCommitStatus struct {
	ConnectionId  uint64 `gorm:"primaryKey"`
	GithubId      int64  `gorm:"primaryKey"`                   // GitHub's unique status ID
	RepoId        uint64 `gorm:"index"`                        // Repository ID for filtering
	CommitSha     string `gorm:"index;type:varchar(40)"`       // Index for querying by commit
	Context       string `gorm:"index;type:varchar(255)"`      // The status context (e.g., "continuous-integration/jenkins")
	State         string `gorm:"type:varchar(100)"`            // Status state: error, failure, pending, success
	Description   string `gorm:"type:text"`                    // Short description of the status
	TargetUrl     string `gorm:"type:varchar(255)"`            // URL to see more details about the status
	AvatarUrl     string `gorm:"type:varchar(255)"`            // Avatar URL of the status creator
	CreatorId     int64  // GitHub user ID of the creator
	CreatorLogin  string `gorm:"type:varchar(255)"` // Login name of the creator
	GithubCreatedAt time.Time
	GithubUpdatedAt time.Time
	common.NoPKModel
}

func (GithubCommitStatus) TableName() string {
	return "_tool_github_commit_statuses"
}
