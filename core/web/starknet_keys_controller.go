package web

import (
	"github.com/pickleyd/chainlink/core/services/chainlink"
	"github.com/pickleyd/chainlink/core/web/presenters"
	starkkey "github.com/smartcontractkit/chainlink-starknet/relayer/pkg/chainlink/keys"
)

func NewStarkNetKeysController(app chainlink.Application) KeysController {
	return NewKeysController[starkkey.Key, presenters.StarkNetKeyResource](app.GetKeyStore().StarkNet(), app.GetLogger(),
		"starknetKey", presenters.NewStarkNetKeyResource, presenters.NewStarkNetKeyResources)
}
