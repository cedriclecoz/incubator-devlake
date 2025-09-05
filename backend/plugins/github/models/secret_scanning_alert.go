package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

// SecretScanningAlert represents a GitHub Advanced Security Secret Scanning alert
// See: https://docs.github.com/en/rest/secret-scanning?apiVersion=2022-11-28

type GithubSecretScanningAlert struct {
		common.Model
		common.RawDataOrigin

	RepoId      string     `gorm:"index;type:varchar(255)" json:"repo_id"`
	AlertNumber int        `gorm:"index" json:"alert_number"`
	SecretType  string     `gorm:"type:varchar(255)" json:"secret_type"`
	State       string     `gorm:"type:varchar(50)" json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	ResolvedBy  string     `gorm:"type:varchar(255)" json:"resolved_by"`
	Resolution  string     `gorm:"type:varchar(255)" json:"resolution"`
	Location    string     `gorm:"type:text" json:"location"`
	Secret      string     `gorm:"type:text" json:"secret"`

	Url                                 string     `gorm:"type:text" json:"url"`
	HtmlUrl                             string     `gorm:"type:text" json:"html_url"`
	LocationsUrl                        string     `gorm:"type:text" json:"locations_url"`
	SecretTypeDisplayName               string     `gorm:"type:varchar(255)" json:"secret_type_display_name"`
	Validity                            string     `gorm:"type:varchar(50)" json:"validity"`
	MultiRepo                           bool       `json:"multi_repo"`
	IsBase64Encoded                     bool       `json:"is_base64_encoded"`
	HasMoreLocations                    bool       `json:"has_more_locations"`
	PubliclyLeaked                      bool       `json:"publicly_leaked"`
	ResolutionComment                   string     `gorm:"type:text" json:"resolution_comment"`
	PushProtectionBypassed              bool       `json:"push_protection_bypassed"`
	PushProtectionBypassedBy            string     `gorm:"type:varchar(255)" json:"push_protection_bypassed_by"`
	PushProtectionBypassedAt            *time.Time `json:"push_protection_bypassed_at"`
	PushProtectionBypassRequestReviewer string     `gorm:"type:varchar(255)" json:"push_protection_bypass_request_reviewer"`
	PushProtectionBypassRequestReviewerComment string `gorm:"type:text" json:"push_protection_bypass_request_reviewer_comment"`
	PushProtectionBypassRequestComment  string     `gorm:"type:text" json:"push_protection_bypass_request_comment"`
	PushProtectionBypassRequestHtmlUrl  string     `gorm:"type:text" json:"push_protection_bypass_request_html_url"`
}

func (GithubSecretScanningAlert) TableName() string {
	return "github_secret_scanning_alerts"
}
