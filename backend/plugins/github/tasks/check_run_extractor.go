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
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

func init() {
	RegisterSubtaskMeta(&ExtractApiCheckRunsMeta)
}

var ExtractApiCheckRunsMeta = plugin.SubTaskMeta{
	Name:             "Extract Check Runs",
	EntryPoint:       ExtractApiCheckRuns,
	EnabledByDefault: true,
	Description:      "Extract raw check runs data into tool layer table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD, plugin.DOMAIN_TYPE_CODE_REVIEW},
	DependencyTables: []string{RAW_CHECK_RUN_TABLE},
	ProductTables:    []string{models.GithubCheckRun{}.TableName()},
}

type ApiCheckRunResponse struct {
	Id          int64      `json:"id"`
	HeadSha     string     `json:"head_sha"`
	ExternalId  string     `json:"external_id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Conclusion  string     `json:"conclusion"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	DetailsUrl  string     `json:"details_url"`
	HtmlUrl     string     `json:"html_url"`
	CheckSuite  struct {
		Id int64 `json:"id"`
	} `json:"check_suite"`
	App struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"app"`
}

func ExtractApiCheckRuns(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*GithubTaskData)

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: GithubApiParams{
				ConnectionId: data.Options.ConnectionId,
				Name:         data.Options.Name,
			},
			Table: RAW_CHECK_RUN_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			body := &ApiCheckRunResponse{}
			if strings.HasPrefix(string(row.Data), "{\"message\": \"Not Found\"") {
				return nil, nil
			}
			err := errors.Convert(json.Unmarshal(row.Data, body))
			if err != nil {
				return nil, err
			}

			checkRun := &models.GithubCheckRun{
				ConnectionId:      data.Options.ConnectionId,
				GithubId:          body.Id,
				RepoId:            uint64(data.Options.GithubId),
				HeadSha:           body.HeadSha,
				ExternalId:        body.ExternalId,
				Name:              body.Name,
				Status:            body.Status,
				Conclusion:        body.Conclusion,
				DetailsUrl:        body.DetailsUrl,
				HtmlUrl:           body.HtmlUrl,
				CheckSuiteId:      body.CheckSuite.Id,
				AppId:             body.App.Id,
				AppName:           body.App.Name,
				AppSlug:           body.App.Slug,
				GithubStartedAt:   body.StartedAt,
				GithubCompletedAt: body.CompletedAt,
			}

			results := make([]interface{}, 0, 1)
			results = append(results, checkRun)

			return results, nil
		},
	})

	if err != nil {
		return err
	}

	return extractor.Execute()
}
