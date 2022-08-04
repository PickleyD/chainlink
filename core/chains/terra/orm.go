package terra

import (
	"github.com/smartcontractkit/sqlx"

	terradb "github.com/smartcontractkit/chainlink-terra/pkg/terra/db"

	"github.com/pickleyd/chainlink/core/chains"
	"github.com/pickleyd/chainlink/core/chains/terra/types"
	"github.com/pickleyd/chainlink/core/logger"
	"github.com/pickleyd/chainlink/core/services/pg"
)

// NewORM returns an ORM backed by db.
func NewORM(db *sqlx.DB, lggr logger.Logger, cfg pg.LogConfig) types.ORM {
	q := pg.NewQ(db, lggr.Named("ORM"), cfg)
	return chains.NewORM[string, *terradb.ChainCfg, terradb.Node](q, "terra", "tendermint_url")
}
