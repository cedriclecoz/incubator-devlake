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
	"fmt"
	"reflect"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/devops"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

func init() {
	RegisterSubtaskMeta(&ConvertCommitStatusesMeta)
}

var ConvertCommitStatusesMeta = plugin.SubTaskMeta{
	Name:             "Convert Commit Statuses",
	EntryPoint:       ConvertCommitStatuses,
	EnabledByDefault: true,
	Description:      "Convert tool layer table github_commit_statuses into domain layer table cicd_pipeline_commits",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD, plugin.DOMAIN_TYPE_CODE},
	DependencyTables: []string{
		models.GithubCommitStatus{}.TableName(),
		models.GithubRepo{}.TableName(),
	},
	ProductTables: []string{devops.CiCDPipelineCommit{}.TableName()},
}

func ConvertCommitStatuses(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*GithubTaskData)

	// Query to get only the latest status for each commit+context combination for this repo
	// Use window function for better performance than GROUP BY + JOIN
	cursor, err := db.Cursor(
		dal.From(&models.GithubCommitStatus{}),
		dal.Join(`INNER JOIN (
			SELECT github_id, commit_sha, context,
				ROW_NUMBER() OVER (
					PARTITION BY commit_sha, context
					ORDER BY github_updated_at DESC, github_id DESC
				) as rn
			FROM _tool_github_commit_statuses
			WHERE connection_id = ? AND repo_id = ?
		) latest ON _tool_github_commit_statuses.github_id = latest.github_id
			AND latest.rn = 1`,
			data.Options.ConnectionId, data.Options.GithubId),
		dal.Where("_tool_github_commit_statuses.connection_id = ? AND _tool_github_commit_statuses.repo_id = ?",
			data.Options.ConnectionId, data.Options.GithubId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	repoIdGen := didgen.NewDomainIdGenerator(&models.GithubRepo{})

	converter, err := api.NewDataConverter(api.DataConverterArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: GithubApiParams{
				ConnectionId: data.Options.ConnectionId,
				Name:         data.Options.Name,
			},
			Table: RAW_COMMIT_STATUS_TABLE,
		},
		InputRowType: reflect.TypeOf(models.GithubCommitStatus{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			commitStatus := inputRow.(*models.GithubCommitStatus)

			// Generate a unique pipeline ID based on connection and context
			// The context represents a unique CI/CD pipeline (e.g., "continuous-integration/jenkins")
			pipelineId := fmt.Sprintf("%s:%d:%s", "github", data.Options.ConnectionId, commitStatus.Context)

			domainPipelineCommit := &devops.CiCDPipelineCommit{
				PipelineId:   pipelineId,
				CommitSha:    commitStatus.CommitSha,
				CommitMsg:    commitStatus.Description,
				DisplayTitle: commitStatus.Context,
				Url:          commitStatus.TargetUrl,
				Branch:       "",
				RepoId:       repoIdGen.Generate(data.Options.ConnectionId, data.Options.GithubId),
				RepoUrl:      "",
			}

			return []interface{}{domainPipelineCommit}, nil
		},
	})

	if err != nil {
		return err
	}

	return converter.Execute()
}
