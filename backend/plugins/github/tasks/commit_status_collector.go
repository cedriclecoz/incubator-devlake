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
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/core/plugin"
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
	DependencyTables: []string{RAW_PULL_REQUEST_TABLE, RAW_PR_COMMIT_TABLE},
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

	// Query commits from recently updated PRs for truly incremental collection
	// Use the incremental time window to filter PR commits, matching the PR commit collector's behavior
	rawDataParams := fmt.Sprintf(`{"ConnectionId":%d,"Name":"%s"}`, data.Options.ConnectionId, data.Options.Name)

	// Check if we have any existing commit statuses for this repo
	// If not, do a full collection even if incremental mode is enabled
	var existingCount int64
	existingCount, err = db.Count(dal.From("_tool_github_commit_statuses"),
		dal.Where("connection_id = ? AND repo_id = ?", data.Options.ConnectionId, data.Options.GithubId))
	if err != nil {
		return err
	}

	var cursor *sql.Rows
	// If incremental AND we already have data, only get commits from the current incremental window
	if apiCollector.IsIncremental() && apiCollector.GetSince() != nil && existingCount > 0 {
		cursor, err = db.RawCursor(`
			SELECT DISTINCT JSON_UNQUOTE(JSON_EXTRACT(CONVERT(prc.data USING utf8mb4), '$.sha')) as commit_sha
			FROM _raw_github_api_pull_request_commits prc
			WHERE prc.params = ?
				AND prc.created_at >= ?
		`, rawDataParams, apiCollector.GetSince())
		taskCtx.GetLogger().Info("Incremental collection: querying commits since %v", apiCollector.GetSince())
	} else {
		// Full collection: get all PR commits for this repo (first run or forced full)
		cursor, err = db.RawCursor(`
			SELECT DISTINCT JSON_UNQUOTE(JSON_EXTRACT(CONVERT(data USING utf8mb4), '$.sha')) as commit_sha
			FROM _raw_github_api_pull_request_commits
			WHERE params = ?
		`, rawDataParams)
		if existingCount == 0 {
			taskCtx.GetLogger().Info("First run detected (no existing data), doing full collection for all PR commits")
		} else {
			taskCtx.GetLogger().Info("Full collection mode")
		}
	}
	if err != nil {
		return err
	}

	taskCtx.GetLogger().Info("Querying PR commits for statuses from raw data in repo_id=%d, connection_id=%d, incremental=%v",
		data.Options.GithubId, data.Options.ConnectionId, apiCollector.IsIncremental())

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
