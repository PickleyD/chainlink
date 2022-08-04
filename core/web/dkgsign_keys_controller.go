package web

import (
	"github.com/pickleyd/chainlink/core/services/chainlink"
	"github.com/pickleyd/chainlink/core/services/keystore/keys/dkgsignkey"
	"github.com/pickleyd/chainlink/core/web/presenters"
)

func NewDKGSignKeysController(app chainlink.Application) KeysController {
	return NewKeysController[dkgsignkey.Key, presenters.DKGSignKeyResource](
		app.GetKeyStore().DKGSign(),
		app.GetLogger(),
		"dkgsignKey",
		presenters.NewDKGSignKeyResource,
		presenters.NewDKGSignKeyResources)
}
