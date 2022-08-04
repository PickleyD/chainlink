package web

import (
	"github.com/pickleyd/chainlink/core/services/chainlink"
	"github.com/pickleyd/chainlink/core/services/keystore/keys/solkey"
	"github.com/pickleyd/chainlink/core/web/presenters"
)

func NewSolanaKeysController(app chainlink.Application) KeysController {
	return NewKeysController[solkey.Key, presenters.SolanaKeyResource](app.GetKeyStore().Solana(), app.GetLogger(),
		"solanaKey", presenters.NewSolanaKeyResource, presenters.NewSolanaKeyResources)
}
