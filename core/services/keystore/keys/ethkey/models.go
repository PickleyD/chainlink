package ethkey

import (
	"time"

	"github.com/smartcontractkit/chainlink/core/utils"
)

type State struct {
	ID         int32
	Address    EIP55Address
	Enabled    bool
	EVMChainID utils.Big
	// NextNonce is only used for initial population and should not be relied
	// on as up-to-date. Nonce source of truth is _always_ the database
	NextNonce int64
	Disabled  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	lastUsed  time.Time
}

func (s State) KeyID() string {
	return s.Address.Hex()
}

// lastUsed is an internal field and ought not be persisted to the database or
// exposed outside of the application
func (s State) LastUsed() time.Time {
	return s.lastUsed
}

func (s *State) WasUsed() {
	s.lastUsed = time.Now()
}
