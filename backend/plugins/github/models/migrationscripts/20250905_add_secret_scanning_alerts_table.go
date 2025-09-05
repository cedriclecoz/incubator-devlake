package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

type addSecretScanningAlertsTable struct{}

func (u *addSecretScanningAlertsTable) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(basicRes, &models.GithubSecretScanningAlert{})
}

func (*addSecretScanningAlertsTable) Version() uint64 {
	return 20250905100000 // Use current date/time for uniqueness
}

func (*addSecretScanningAlertsTable) Name() string {
	return "Add github_secret_scanning_alerts table"
}
