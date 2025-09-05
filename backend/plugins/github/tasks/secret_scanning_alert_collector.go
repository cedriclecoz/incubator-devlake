package tasks

import (
	"net/url"
	"net/http"
	"time"
	"fmt"
	"encoding/json"
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
			   // clclc: log UrlTemplate, Params, and Pager before each external call
			   taskCtx.GetLogger().Info(fmt.Sprintf(
				   "clclc: [GitHub API Call] UrlTemplate: %s, Params: %+v, Pager: %+v",
				   "repos/{{ .Params.Name }}/secret-scanning/alerts",
				   reqData.Params,
				   reqData.Pager,
			   ))
			   query := url.Values{}
			   if apiCollector.IsIncremental() && apiCollector.GetSince() != nil {
				   query.Set("since", apiCollector.GetSince().Format(time.RFC3339))
			   }
			   return query, nil
		   },
		   ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			   arr, err := helper.GetRawMessageArrayFromResponse(res)
			   taskCtx.GetLogger().Info(fmt.Sprintf("clclc: [GitHub API Call] Got %d results in this page", len(arr)))
			   // Patch: set url field for each alert in the raw table and deduplicate by alert_url
			   seen := make(map[string]struct{})
			   deduped := make([]json.RawMessage, 0, len(arr))
			   for i, raw := range arr {
				   var alert map[string]interface{}
				   if err := json.Unmarshal(raw, &alert); err == nil {
					   // Print the alert for debugging
					   taskCtx.GetLogger().Info(fmt.Sprintf("clclc: [DEBUG] alert: %+v", alert))
					   taskCtx.GetLogger().Info(fmt.Sprintf("clclc: [DEBUG] number: %v", alert["number"]))
					   taskCtx.GetLogger().Info(fmt.Sprintf("clclc: [DEBUG] url: %s", alert["url"]))
					   // Always set alert_url to a non-empty, unique value based on repo and alert number
					   var alertUrl string
					   if number, ok := alert["number"]; ok && fmt.Sprintf("%v", number) != "" {
						   alertUrl = fmt.Sprintf("https://api.github.com/repos/%s/secret-scanning/alerts/%v", data.Options.Name, number)
					   } else {
						   // fallback: use repo + timestamp + random suffix
						   alertUrl = fmt.Sprintf("https://api.github.com/repos/%s/secret-scanning/alerts/fallback-%d-%d", data.Options.Name, time.Now().UnixNano(), i)
					   }
					   alert["alert_url"] = alertUrl
					   if _, exists := seen[alertUrl]; exists {
						   continue // skip duplicate in this batch
					   }
					   seen[alertUrl] = struct{}{}
					   marshaled, _ := json.Marshal(alert)
					   deduped = append(deduped, marshaled)
				   }
			   }
			   return deduped, err
		   },
		   GetTotalPages: func(res *http.Response, args *helper.ApiCollectorArgs) (int, errors.Error) {
			   arr, err := helper.GetRawMessageArrayFromResponse(res)
			   if err != nil {
				   return 0, err
			   }
			   // If less than PageSize, only one page is needed
			   if len(arr) < args.PageSize {
				   return 1, nil
			   }
			   // Otherwise, let the collector try the next page
			   return 2, nil
		   },
	   })
	if err != nil {
		return err
	}

       err = apiCollector.Execute()
       if err != nil {
	       // Ignore duplicate key errors (MySQL error 1062)
	       if isDuplicateKeyError(err) {
		       taskCtx.GetLogger().Warn(nil, "Duplicate alert_url detected, skipping insert.")
		       return nil // continue as success
	       } else {
		       return err
	       }
       }

	db := taskCtx.GetDal()
	var count int64
	// Count collected alerts using DAL's Count method
    count, err = db.Count(dal.From(fmt.Sprintf("_raw_%s", RAW_SECRET_SCANNING_ALERTS_TABLE)))
	if err != nil {
		       taskCtx.GetLogger().Error(err, "Failed to count collected secret scanning alerts")
	       } else {
		       taskCtx.GetLogger().Info(fmt.Sprintf("Collected %d secret scanning alerts", count))
	       }
	return nil
}
