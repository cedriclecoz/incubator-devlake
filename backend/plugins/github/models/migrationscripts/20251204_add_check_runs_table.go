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

type addCheckRunsTable struct{}

type checkRun20251204 struct {
	ConnectionId      uint64 `gorm:"primaryKey"`
	GithubId          int64  `gorm:"primaryKey"`
	RepoId            uint64 `gorm:"index"`
	HeadSha           string `gorm:"index;type:varchar(40)"`
	ExternalId        string `gorm:"type:varchar(255)"`
	Name              string `gorm:"index;type:varchar(255)"`
	Status            string `gorm:"type:varchar(100)"`
	Conclusion        string `gorm:"type:varchar(100)"`
	DetailsUrl        string `gorm:"type:varchar(255)"`
	HtmlUrl           string `gorm:"type:varchar(255)"`
	CheckSuiteId      int64  `gorm:"index"`
	AppId             int64
	AppName           string `gorm:"type:varchar(255)"`
	AppSlug           string `gorm:"type:varchar(255)"`
	GithubStartedAt   *time.Time
	GithubCompletedAt *time.Time
	archived.NoPKModel
}

func (checkRun20251204) TableName() string {
	return "_tool_github_check_runs"
}

func (u *addCheckRunsTable) Up(basicRes context.BasicRes) errors.Error {
	err := migrationhelper.AutoMigrateTables(
		basicRes,
		&checkRun20251204{},
	)
	if err != nil {
		return err
	}

	// Add composite index for optimized GROUP BY queries
	db := basicRes.GetDal()
	err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_check_runs_composite
		ON _tool_github_check_runs(head_sha, connection_id, name, github_completed_at, updated_at, github_id)
	`)
	return err
}

func (*addCheckRunsTable) Version() uint64 {
	return 20251204000001
}

func (*addCheckRunsTable) Name() string {
	return "add _tool_github_check_runs table"
}
