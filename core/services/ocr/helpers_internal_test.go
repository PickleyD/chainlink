package ocr

import (
	"testing"

	"github.com/smartcontractkit/sqlx"

	"github.com/pickleyd/chainlink/core/logger"
	"github.com/pickleyd/chainlink/core/testutils/configtest"
)

func (c *ConfigOverriderImpl) ExportedUpdateFlagsStatus() error {
	return c.updateFlagsStatus()
}

func NewTestDB(t *testing.T, sqldb *sqlx.DB, oracleSpecID int32) *db {
	return NewDB(sqldb, oracleSpecID, logger.TestLogger(t), configtest.NewTestGeneralConfig(t))
}
