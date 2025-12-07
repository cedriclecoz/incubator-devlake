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
	RegisterSubtaskMeta(&CollectApiCheckRunsMeta)
}

const RAW_CHECK_RUN_TABLE = "github_api_check_runs"

var CollectApiCheckRunsMeta = plugin.SubTaskMeta{
	Name:             "Collect Check Runs",
	EntryPoint:       CollectApiCheckRuns,
	EnabledByDefault: true,
	Description:      "Collect check runs data from Github api for PR commits",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD, plugin.DOMAIN_TYPE_CODE_REVIEW},
	DependencyTables: []string{models.GithubPrCommit{}.TableName()},
	ProductTables:    []string{RAW_CHECK_RUN_TABLE},
}

func CollectApiCheckRuns(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*GithubTaskData)

	apiCollector, err := helper.NewStatefulApiCollector(helper.RawDataSubTaskArgs{
		Ctx: taskCtx,
		Params: GithubApiParams{
			ConnectionId: data.Options.ConnectionId,
			Name:         data.Options.Name,
		},
		Table: RAW_CHECK_RUN_TABLE,
	})
	if err != nil {
		return err
	}

	// Query PR commits to get the list of commit SHAs to collect check runs for
	// Collect for open PRs and recently updated closed/merged PRs (to catch status updates)
	clauses := []dal.Clause{
		dal.Select("commit_sha"),
		dal.From(models.GithubPrCommit{}.TableName()),
		dal.Join(`LEFT JOIN _tool_github_pull_requests pr ON
			_tool_github_pull_request_commits.connection_id = pr.connection_id AND
			_tool_github_pull_request_commits.pull_request_id = pr.github_id`),
		dal.Where("_tool_github_pull_request_commits.connection_id = ? AND pr.repo_id = ?",
			data.Options.ConnectionId, data.Options.GithubId),
		dal.Groupby("commit_sha"),
	}

	// If incremental, filter to only open PRs or recently updated ones
	if apiCollector.IsIncremental() && apiCollector.GetSince() != nil {
		clauses = append(clauses,
			dal.Where("pr.state = ? OR pr.github_updated_at > ?", "open", apiCollector.GetSince()),
		)
	} else {
		// On full sync, only collect for open PRs to avoid excessive API calls
		clauses = append(clauses,
			dal.Where("pr.state = ?", "open"),
		)
	}

	taskCtx.GetLogger().Info("Querying PR commits for check runs in repo_id=%d, connection_id=%d, incremental=%v",
		data.Options.GithubId, data.Options.ConnectionId, apiCollector.IsIncremental())

	cursor, err := db.Cursor(clauses...)
	if err != nil {
		return err
	}

	iterator, err := helper.NewDalCursorIterator(db, cursor, reflect.TypeOf(SimpleCommit{}))
	if err != nil {
		return err
	}

	taskCtx.GetLogger().Info("Iterator created for check runs collection")

	err = apiCollector.InitCollector(helper.ApiCollectorArgs{
		ApiClient: data.ApiClient,
		PageSize:  100,
		Input:     iterator,

		UrlTemplate: "repos/{{ .Params.Name }}/commits/{{ .Input.CommitSha }}/check-runs",

		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("page", fmt.Sprintf("%v", reqData.Pager.Page))
			query.Set("per_page", fmt.Sprintf("%v", reqData.Pager.Size))
			return query, nil
		},

		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			// GitHub check-runs API returns {check_runs: [...]} not a direct array
			var body struct {
				CheckRuns []json.RawMessage `json:"check_runs"`
			}
			err := helper.UnmarshalResponse(res, &body)
			if err != nil {
				return nil, err
			}
			return body.CheckRuns, nil
		},
	})

	if err != nil {
		return err
	}

	return apiCollector.Execute()
}
