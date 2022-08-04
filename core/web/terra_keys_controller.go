package web

import (
	"github.com/pickleyd/chainlink/core/services/chainlink"
	"github.com/pickleyd/chainlink/core/services/keystore/keys/terrakey"
	"github.com/pickleyd/chainlink/core/web/presenters"
)

func NewTerraKeysController(app chainlink.Application) KeysController {
	return NewKeysController[terrakey.Key, presenters.TerraKeyResource](app.GetKeyStore().Terra(), app.GetLogger(),
		"terraKey", presenters.NewTerraKeyResource, presenters.NewTerraKeyResources)
}
