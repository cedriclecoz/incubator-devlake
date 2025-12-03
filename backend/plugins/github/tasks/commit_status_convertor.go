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

	// Query to get only the latest status for each commit+context combination
	// We use a subquery to find the max github_updated_at for each commit_sha+context pair
	cursor, err := db.Cursor(
		dal.From(&models.GithubCommitStatus{}),
		dal.Join(`INNER JOIN (
			SELECT commit_sha, context, MAX(github_updated_at) as latest_update
			FROM _tool_github_commit_statuses
			WHERE connection_id = ?
			GROUP BY commit_sha, context
		) latest ON _tool_github_commit_statuses.commit_sha = latest.commit_sha
			AND _tool_github_commit_statuses.context = latest.context
			AND _tool_github_commit_statuses.github_updated_at = latest.latest_update`, data.Options.ConnectionId),
		dal.Where("_tool_github_commit_statuses.connection_id = ?", data.Options.ConnectionId),
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
