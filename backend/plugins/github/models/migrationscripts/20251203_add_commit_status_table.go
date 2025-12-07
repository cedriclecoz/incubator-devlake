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

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/migrationscripts/archived"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type addCommitStatusTable struct{}

type commitStatus20251203 struct {
	ConnectionId  uint64 `gorm:"primaryKey"`
	GithubId      int64  `gorm:"primaryKey"`
	RepoId        uint64 `gorm:"index"`
	CommitSha     string `gorm:"index;type:varchar(40)"`
	Context       string `gorm:"index;type:varchar(255)"`
	State         string `gorm:"type:varchar(100)"`
	Description     string `gorm:"type:text"`
	TargetUrl       string `gorm:"type:varchar(255)"`
	AvatarUrl       string `gorm:"type:varchar(255)"`
	CreatorId       int64
	CreatorLogin    string `gorm:"type:varchar(255)"`
	GithubCreatedAt time.Time
	GithubUpdatedAt time.Time
	archived.NoPKModel
}

func (commitStatus20251203) TableName() string {
	return "_tool_github_commit_statuses"
}

func (u *addCommitStatusTable) Up(basicRes context.BasicRes) errors.Error {
	err := migrationhelper.AutoMigrateTables(
		basicRes,
		&commitStatus20251203{},
	)
	if err != nil {
		return err
	}

	// Add composite index for optimized GROUP BY queries
	db := basicRes.GetDal()
	err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_commit_statuses_composite
		ON _tool_github_commit_statuses(commit_sha, connection_id, context, github_updated_at, github_id)
	`)
	return err
}

func (*addCommitStatusTable) Version() uint64 {
	return 20251203000001
}

func (*addCommitStatusTable) Name() string {
	return "add _tool_github_commit_statuses table"
}
