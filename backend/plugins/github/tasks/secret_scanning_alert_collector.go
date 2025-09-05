package tasks

import (
	"net/url"
	"net/http"
	"time"
	"fmt"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// GithubSecretScanningAlertResponse represents the response from GitHub API
// See: https://docs.github.com/en/rest/secret-scanning?apiVersion=2022-11-28#list-secret-scanning-alerts-for-a-repository

type GithubSecretScanningAlertResponse struct {
	Number     int        `json:"number"`
	SecretType string     `json:"secret_type"`
	State      string     `json:"state"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	ResolvedBy string     `json:"resolved_by"`
	Resolution string     `json:"resolution"`
	Location   string     `json:"location"`
	Secret     string     `json:"secret"`
}

func CollectSecretScanningAlerts(taskCtx plugin.SubTaskContext) errors.Error {
		taskCtx.GetLogger().Info("clclc: Entered CollectSecretScanningAlerts entrypoint")
		data := taskCtx.GetData().(*GithubTaskData)
	const RAW_SECRET_SCANNING_ALERTS_TABLE = "github_api_secret_scanning_alerts"

	apiCollector, err := helper.NewStatefulApiCollector(helper.RawDataSubTaskArgs{
		Ctx: taskCtx,
		Params: GithubApiParams{
			ConnectionId: data.Options.ConnectionId,
			Name:         data.Options.Name,
		},
		Table: RAW_SECRET_SCANNING_ALERTS_TABLE,
	})
	if err != nil {
		return err
	}

	   err = apiCollector.InitCollector(helper.ApiCollectorArgs{
		   ApiClient:   data.ApiClient,
		   PageSize:    100,
		   UrlTemplate: "repos/{{ .Params.Name }}/secret-scanning/alerts",
		   Header: func(reqData *helper.RequestData) (http.Header, errors.Error) {
			   header := http.Header{}
			   header.Set("Accept", "application/vnd.github+json")
			   return header, nil
		   },
		   Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			   query := url.Values{}
			   if apiCollector.IsIncremental() && apiCollector.GetSince() != nil {
				   query.Set("since", apiCollector.GetSince().Format(time.RFC3339))
			   }
			   return query, nil
		   },
		   ResponseParser: helper.GetRawMessageArrayFromResponse,
	   })
	if err != nil {
		return err
	}

	err = apiCollector.Execute()
	if err != nil {
		return err
	}

	// Log what was collected (clclc)
	db := taskCtx.GetDal()
	var count int64
	// Count collected alerts using DAL's Count method
	count, err = db.Count(dal.From(RAW_SECRET_SCANNING_ALERTS_TABLE))
	if err != nil {
		taskCtx.GetLogger().Error(err, "clclc: Failed to count collected secret scanning alerts")
	} else {
		taskCtx.GetLogger().Info(fmt.Sprintf("clclc: Collected %d secret scanning alerts", count))
	}
	return nil
}
