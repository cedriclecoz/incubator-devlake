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

package tasks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/dal"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

func init() {
	RegisterSubtaskMeta(&CollectApiCommitStatusesMeta)
}

const RAW_COMMIT_STATUS_TABLE = "github_api_commit_statuses"

var CollectApiCommitStatusesMeta = plugin.SubTaskMeta{
	Name:             "Collect Commit Statuses",
	EntryPoint:       CollectApiCommitStatuses,
	EnabledByDefault: true,
	Description:      "Collect commit status checks data from Github api for PR commits",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD, plugin.DOMAIN_TYPE_CODE_REVIEW},
	DependencyTables: []string{models.GithubPrCommit{}.TableName()},
	ProductTables:    []string{RAW_COMMIT_STATUS_TABLE},
}

type SimpleCommit struct {
	CommitSha string
}

func CollectApiCommitStatuses(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*GithubTaskData)

	apiCollector, err := helper.NewStatefulApiCollector(helper.RawDataSubTaskArgs{
		Ctx: taskCtx,
		Params: GithubApiParams{
			ConnectionId: data.Options.ConnectionId,
			Name:         data.Options.Name,
		},
		Table: RAW_COMMIT_STATUS_TABLE,
	})
	if err != nil {
		return err
	}

	// Query PR commits to get the list of commit SHAs to collect statuses for
	// We want to collect statuses for all PR commits, not just recently updated PRs,
	// because commit statuses can change independently of PR updates (CI/CD re-runs).
	// Also, we want to collect statuses for open PRs to show current status in dashboards.
	clauses := []dal.Clause{
		dal.Select("commit_sha"),
		dal.From(models.GithubPrCommit{}.TableName()),
		dal.Join(`LEFT JOIN _tool_github_pull_requests pr ON
			_tool_github_pull_request_commits.connection_id = pr.connection_id AND
			_tool_github_pull_request_commits.pull_request_id = pr.github_id`),
		dal.Where("_tool_github_pull_request_commits.connection_id = ? AND pr.repo_id = ? AND pr.state = ?",
			data.Options.ConnectionId, data.Options.GithubId, "open"),
		dal.Groupby("commit_sha"),
	}

	taskCtx.GetLogger().Info("Querying PR commits for open PRs in repo_id=%d, connection_id=%d", data.Options.GithubId, data.Options.ConnectionId)

	cursor, err := db.Cursor(clauses...)
	if err != nil {
		return err
	}

	iterator, err := helper.NewDalCursorIterator(db, cursor, reflect.TypeOf(SimpleCommit{}))
	if err != nil {
		return err
	}

	// Debug: Log first few items from iterator to verify it's working
	taskCtx.GetLogger().Info("Iterator created for commit status collection")

	err = apiCollector.InitCollector(helper.ApiCollectorArgs{
		ApiClient: data.ApiClient,
		PageSize:  100,
		Input:     iterator,

		UrlTemplate: "repos/{{ .Params.Name }}/commits/{{ .Input.CommitSha }}/statuses",

		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("page", fmt.Sprintf("%v", reqData.Pager.Page))
			query.Set("per_page", fmt.Sprintf("%v", reqData.Pager.Size))
			return query, nil
		},

		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var items []json.RawMessage
			err := helper.UnmarshalResponse(res, &items)
			if err != nil {
				return nil, err
			}
			return items, nil
		},
	})

	if err != nil {
		return err
	}

	return apiCollector.Execute()
}
