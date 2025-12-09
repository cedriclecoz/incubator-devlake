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
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

func init() {
	RegisterSubtaskMeta(&CollectJobsMeta)
}

const RAW_JOB_TABLE = "github_api_jobs"

var CollectJobsMeta = plugin.SubTaskMeta{
	Name:             "Collect Job Runs",
	EntryPoint:       CollectJobs,
	EnabledByDefault: true,
	Description:      "Collect Jobs data from Github action api, supports both timeFilter and diffSync.",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
	DependencyTables: []string{RAW_RUN_TABLE},
	ProductTables:    []string{RAW_JOB_TABLE},
}

func CollectJobs(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*GithubTaskData)

	// state manager
	apiCollector, err := api.NewStatefulApiCollector(api.RawDataSubTaskArgs{
		Ctx: taskCtx,
		Params: GithubApiParams{
			ConnectionId: data.Options.ConnectionId,
			Name:         data.Options.Name,
		},
		Table: RAW_JOB_TABLE,
	})
	if err != nil {
		return err
	}

	// Query workflow runs from raw data table (collected in this run) instead of the GithubRun table
	// This ensures we only collect jobs for runs that were actually collected/updated in this execution
	// Use raw SQL to extract ID from JSON data field
	rawDataParams := fmt.Sprintf(`{"ConnectionId":%d,"Name":"%s"}`, data.Options.ConnectionId, data.Options.Name)
	cursor, err := db.RawCursor(`
		SELECT DISTINCT CAST(JSON_EXTRACT(data, '$.id') AS SIGNED) as id
		FROM _raw_github_api_runs
		WHERE params = ?
	`, rawDataParams)
	if err != nil {
		return err
	}

	iterator, err := api.NewDalCursorIterator(db, cursor, reflect.TypeOf(SimpleGithubRun{}))
	if err != nil {
		return err
	}
	// collect jobs
	err = apiCollector.InitCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: GithubApiParams{
				ConnectionId: data.Options.ConnectionId,
				Name:         data.Options.Name,
			},
			Table: RAW_JOB_TABLE,
		},
		ApiClient:   data.ApiClient,
		PageSize:    100,
		Input:       iterator,
		UrlTemplate: "repos/{{ .Params.Name }}/actions/runs/{{ .Input.ID }}/jobs",
		Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("page", fmt.Sprintf("%v", reqData.Pager.Page))
			query.Set("per_page", fmt.Sprintf("%v", reqData.Pager.Size))
			return query, nil
		},
		GetTotalPages: GetTotalPagesFromResponse,
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			body := &GithubRawJobsResult{}
			err := api.UnmarshalResponse(res, body)
			if err != nil {
				return nil, err
			}
			return body.GithubWorkflowJobs, nil
		},
		AfterResponse: ignoreHTTPStatus404,
	})
	if err != nil {
		return err
	}
	return apiCollector.Execute()
}

type SimpleGithubRun struct {
	ID int64
}

type GithubRawJobsResult struct {
	TotalCount         int64             `json:"total_count"`
	GithubWorkflowJobs []json.RawMessage `json:"jobs"`
}
