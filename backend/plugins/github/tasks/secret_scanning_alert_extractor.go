package tasks

import (
	"encoding/json"
	"time"
	"fmt"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	api "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

const RAW_SECRET_SCANNING_ALERTS_TABLE = "github_api_secret_scanning_alerts"

func ExtractSecretScanningAlerts(taskCtx plugin.SubTaskContext) errors.Error {
		taskCtx.GetLogger().Info("clclc: Entered ExtractSecretScanningAlerts entrypoint0")
			extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: GithubApiParams{
				ConnectionId: taskCtx.GetData().(*GithubTaskData).Options.ConnectionId,
				Name:         taskCtx.GetData().(*GithubTaskData).Options.Name,
			},
			Table: RAW_SECRET_SCANNING_ALERTS_TABLE,
		},
		   Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
			   fmt.Println("clclclc Extractor called")
			   // Decode hex if needed, then unmarshal
			   var jsonBytes []byte
			   if len(row.Data) > 2 && row.Data[0] == '0' && row.Data[1] == 'x' {
				   // Hex string, decode
				   hexStr := string(row.Data[2:])
				   decoded := make([]byte, len(hexStr)/2)
				   for i := 0; i < len(decoded); i++ {
					   var b byte
					   _, err := fmt.Sscanf(hexStr[2*i:2*i+2], "%02x", &b)
					   if err != nil {
						   fmt.Printf("[ERROR] clclcl Failed to decode hex data at row %d: %v\n", i, err)
						   return nil, errors.Default.Wrap(err, "failed to decode hex data")
					   }
					   decoded[i] = b
				   }
				   jsonBytes = decoded
			   } else {
				   jsonBytes = row.Data
			   }
			   var data map[string]interface{}
			   if err := json.Unmarshal(jsonBytes, &data); err != nil {
				   fmt.Printf("[ERROR] clclcl Failed to unmarshal JSON: %v\nRaw: %s\n", err, string(jsonBytes))
				   return nil, errors.Default.Wrap(err, "failed to unmarshal secret scanning alert raw data")
			   }

			   alert := &models.GithubSecretScanningAlert{}
			   // Try to extract from data["url"]
			   repoName := ""
			   if urlStr, ok := data["url"].(string); ok {
				   // Example: https://api.github.com/repos/rdkcentral/RDKM-Project-Roar/secret-scanning/alerts/4
				   parts := strings.Split(urlStr, "/")
				   if len(parts) >= 6 {
					   repoName = parts[5] // RDKM-Project-Roar
					   fmt.Printf("clclcl  Extracted alert1: repoName:%s\n", repoName)
				   }
			   }

			   if repoName == "" {
				   repoName = "unknown"
			   }
			   alert.RepoId = repoName
			   // Always set AlertNumber from raw JSON
			   if v, ok := data["number"].(float64); ok {
				   alert.AlertNumber = int(v)
			   } else if v, ok := data["number"].(int); ok {
				   alert.AlertNumber = v
			   } else {
				   alert.AlertNumber = -1 // fallback for missing value
			   }
			   if v, ok := data["secret_type"].(string); ok {
				   alert.SecretType = v
			   }
			   if v, ok := data["state"].(string); ok {
				   alert.State = v
			   }
			   if v, ok := data["created_at"].(string); ok {
				   t, err := time.Parse(time.RFC3339, v)
				   if err == nil {
					   alert.CreatedAt = t
				   }
			   }
			   if v, ok := data["updated_at"].(string); ok {
				   t, err := time.Parse(time.RFC3339, v)
				   if err == nil {
					   alert.UpdatedAt = t
				   }
			   }
			   if v, ok := data["resolved_at"].(string); ok && v != "" {
				   t, err := time.Parse(time.RFC3339, v)
				   if err == nil {
					   alert.ResolvedAt = &t
				   }
			   }
			   if v, ok := data["resolved_by"].(string); ok {
				   alert.ResolvedBy = v
			   }
			   if v, ok := data["resolution"].(string); ok {
				   alert.Resolution = v
			   }
			   if v, ok := data["secret"].(string); ok {
				   alert.Secret = v
			   }
			   // Flatten location info as string
			   if v, ok := data["first_location_detected"].(map[string]interface{}); ok {
				   locBytes, err := json.Marshal(v)
				   if err == nil {
					   alert.Location = string(locBytes)
				   }
			   }
			   if v, ok := data["url"].(string); ok {
				   alert.Url = v
			   }
			   if v, ok := data["html_url"].(string); ok {
				   alert.HtmlUrl = v
			   }
			   if v, ok := data["locations_url"].(string); ok {
				   alert.LocationsUrl = v
			   }
			   if v, ok := data["secret_type_display_name"].(string); ok {
				   alert.SecretTypeDisplayName = v
			   }
			   if v, ok := data["validity"].(string); ok {
				   alert.Validity = v
			   }
			   if v, ok := data["multi_repo"].(bool); ok {
				   alert.MultiRepo = v
			   }
			   if v, ok := data["is_base64_encoded"].(bool); ok {
				   alert.IsBase64Encoded = v
			   }
			   if v, ok := data["has_more_locations"].(bool); ok {
				   alert.HasMoreLocations = v
			   }
			   if v, ok := data["publicly_leaked"].(bool); ok {
				   alert.PubliclyLeaked = v
			   }
			   if v, ok := data["resolution_comment"].(string); ok {
				   alert.ResolutionComment = v
			   }
			   if v, ok := data["push_protection_bypassed"].(bool); ok {
				   alert.PushProtectionBypassed = v
			   }
			   if v, ok := data["push_protection_bypassed_by"].(string); ok {
				   alert.PushProtectionBypassedBy = v
			   }
			   if v, ok := data["push_protection_bypassed_at"].(string); ok && v != "" {
				   t, err := time.Parse(time.RFC3339, v)
				   if err == nil {
					   alert.PushProtectionBypassedAt = &t
				   }
			   }
			   if v, ok := data["push_protection_bypass_request_reviewer"].(string); ok {
				   alert.PushProtectionBypassRequestReviewer = v
			   }
			   if v, ok := data["push_protection_bypass_request_reviewer_comment"].(string); ok {
				   alert.PushProtectionBypassRequestReviewerComment = v
			   }
			   if v, ok := data["push_protection_bypass_request_comment"].(string); ok {
				   alert.PushProtectionBypassRequestComment = v
			   }
			   if v, ok := data["push_protection_bypass_request_html_url"].(string); ok {
				   alert.PushProtectionBypassRequestHtmlUrl = v
			   }
			   // Log the alert's primary keys before returning
			   fmt.Printf("clclc  ALERT TO UPSERT: repo_id=%s, alert_number=%d\n", alert.RepoId, alert.AlertNumber)
			   fmt.Printf("clclc  ALERT STRUCT: %+v\n", alert)
			   // Log the number of alerts being returned (always 1 here, but for consistency)
			   fmt.Printf("clclc  Extract function returning %d alert(s)\n", 1)
			   return []interface{}{alert}, nil
		   },
	})
	if err != nil {
		return err
	}
		// Add detailed logging before and after Execute (batch save/upsert)
		fmt.Printf("clclcl: About to call extractor.Execute (batch save/upsert)\n")
		result := extractor.Execute()
		if result != nil {
			fmt.Printf("clclcl: extractor.Execute returned error: %v\n", result)
		} else {
			fmt.Printf("clclcl: extractor.Execute completed successfully\n")
		}
		fmt.Printf("clclcl: ExtractSecretScanningAlerts exit, return value: %v\n", result)
		return result
}
