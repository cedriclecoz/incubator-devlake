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

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

func init() {
	RegisterSubtaskMeta(&ExtractApiCommitStatusesMeta)
}

var ExtractApiCommitStatusesMeta = plugin.SubTaskMeta{
	Name:             "Extract Commit Statuses",
	EntryPoint:       ExtractApiCommitStatuses,
	EnabledByDefault: true,
	Description:      "Extract raw commit status data into tool layer table github_commit_statuses",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD, plugin.DOMAIN_TYPE_CODE_REVIEW},
	DependencyTables: []string{RAW_COMMIT_STATUS_TABLE},
	ProductTables:    []string{models.GithubCommitStatus{}.TableName()},
}

type CommitStatusResponse struct {
	Id          int64              `json:"id"`
	State       string             `json:"state"`
	Description *string            `json:"description"`
	TargetUrl   *string            `json:"target_url"`
	Context     string             `json:"context"`
	AvatarUrl   *string            `json:"avatar_url"`
	CreatedAt   common.Iso8601Time `json:"created_at"`
	UpdatedAt   common.Iso8601Time `json:"updated_at"`
	Creator     *struct {
		Id        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarUrl string `json:"avatar_url"`
	} `json:"creator"`
}

func ExtractApiCommitStatuses(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*GithubTaskData)

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: GithubApiParams{
				ConnectionId: data.Options.ConnectionId,
				Name:         data.Options.Name,
			},
			Table: RAW_COMMIT_STATUS_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			apiCommitStatus := &CommitStatusResponse{}
			if strings.HasPrefix(string(row.Data), "{\"message\": \"Not Found\"") {
				return nil, nil
			}
			err := errors.Convert(json.Unmarshal(row.Data, apiCommitStatus))
			if err != nil {
				return nil, err
			}

			commit := &SimpleCommit{}
			err = errors.Convert(json.Unmarshal(row.Input, commit))
			if err != nil {
				return nil, err
			}

			// Handle nullable fields
			description := ""
			if apiCommitStatus.Description != nil {
				description = *apiCommitStatus.Description
			}

			targetUrl := ""
			if apiCommitStatus.TargetUrl != nil {
				targetUrl = *apiCommitStatus.TargetUrl
			}

			avatarUrl := ""
			if apiCommitStatus.AvatarUrl != nil {
				avatarUrl = *apiCommitStatus.AvatarUrl
			}

			creatorId := int64(0)
			creatorLogin := ""
			if apiCommitStatus.Creator != nil {
				creatorId = apiCommitStatus.Creator.Id
				creatorLogin = apiCommitStatus.Creator.Login
			}

			githubCommitStatus := &models.GithubCommitStatus{
				ConnectionId:    data.Options.ConnectionId,
				GithubId:        apiCommitStatus.Id,
				RepoId:          uint64(data.Options.GithubId),
				CommitSha:       commit.CommitSha,
				Context:         apiCommitStatus.Context,
				State:           apiCommitStatus.State,
				Description:     description,
				TargetUrl:       targetUrl,
				AvatarUrl:       avatarUrl,
				CreatorId:       creatorId,
				CreatorLogin:    creatorLogin,
				GithubCreatedAt: apiCommitStatus.CreatedAt.ToTime(),
				GithubUpdatedAt: apiCommitStatus.UpdatedAt.ToTime(),
			}

			results := make([]interface{}, 0, 1)
			results = append(results, githubCommitStatus)

			return results, nil
		},
	})

	if err != nil {
		return err
	}

	return extractor.Execute()
}
