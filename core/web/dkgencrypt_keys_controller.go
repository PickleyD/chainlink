package web

import (
	"github.com/pickleyd/chainlink/core/services/chainlink"
	"github.com/pickleyd/chainlink/core/services/keystore/keys/dkgencryptkey"
	"github.com/pickleyd/chainlink/core/web/presenters"
)

func NewDKGEncryptKeysController(app chainlink.Application) KeysController {
	return NewKeysController[dkgencryptkey.Key, presenters.DKGEncryptKeyResource](
		app.GetKeyStore().DKGEncrypt(),
		app.GetLogger(),
		"dkgencryptKey",
		presenters.NewDKGEncryptKeyResource,
		presenters.NewDKGEncryptKeyResources)
}
