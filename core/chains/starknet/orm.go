package starknet

import (
	"github.com/smartcontractkit/sqlx"

	starknetdb "github.com/smartcontractkit/chainlink-starknet/relayer/pkg/chainlink/db"

	"github.com/pickleyd/chainlink/core/chains"
	"github.com/pickleyd/chainlink/core/chains/starknet/types"
	"github.com/pickleyd/chainlink/core/logger"
	"github.com/pickleyd/chainlink/core/services/pg"
)

func NewORM(db *sqlx.DB, lggr logger.Logger, cfg pg.LogConfig) types.ORM {
	q := pg.NewQ(db, lggr.Named("ORM"), cfg)
	return chains.NewORM[string, *starknetdb.ChainCfg, starknetdb.Node](q, "starknet", "url")
}
