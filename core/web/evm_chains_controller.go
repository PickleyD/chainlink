package web

import (
	"github.com/pickleyd/chainlink/core/chains/evm/types"
	"github.com/pickleyd/chainlink/core/services/chainlink"
	"github.com/pickleyd/chainlink/core/utils"
	"github.com/pickleyd/chainlink/core/web/presenters"
)

var ErrEVMNotEnabled = errChainDisabled{name: "EVM", envVar: "EVM_ENABLED"}

func NewEVMChainsController(app chainlink.Application) ChainsController {
	parse := func(s string) (id utils.Big, err error) {
		err = id.UnmarshalText([]byte(s))
		return
	}
	return newChainsController[utils.Big, *types.ChainCfg, presenters.EVMChainResource](
		"evm", app.GetChains().EVM, ErrEVMNotEnabled, parse, presenters.NewEVMChainResource)
}
