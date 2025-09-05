package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
)

type addRawGithubApiSecretScanningAlertsTable struct{}

func (u *addRawGithubApiSecretScanningAlertsTable) Up(basicRes context.BasicRes) errors.Error {
       db := basicRes.GetDal()
       if err := errors.Convert(db.Exec(`
	       CREATE TABLE IF NOT EXISTS _raw_github_api_secret_scanning_alerts (
		       id BIGINT AUTO_INCREMENT PRIMARY KEY,
		       data LONGTEXT NOT NULL,
		       url VARCHAR(512) NOT NULL,
			   alert_url VARCHAR(512) NOT NULL,
		       params JSON,
		       input JSON,
		       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	       )
       `)); err != nil {
	       return err
       }
       return errors.Convert(db.Exec(`
	       ALTER TABLE _raw_github_api_secret_scanning_alerts
	       ADD UNIQUE INDEX uniq_raw_secret_alert_url (alert_url(255))
       `))
}

func (*addRawGithubApiSecretScanningAlertsTable) Version() uint64 {
	return 20250905101000 // Just after the domain table migration
}

func (*addRawGithubApiSecretScanningAlertsTable) Name() string {
	return "Add _raw_github_api_secret_scanning_alerts table with unique url constraint"
}
