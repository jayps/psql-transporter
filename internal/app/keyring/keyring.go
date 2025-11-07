package keyring

import (
	"github.com/99designs/keyring"
)

type Store interface {
	Get(key string) (keyring.Item, error)
	Set(item keyring.Item) error
	Delete(key string) error
}

type store struct{ kr keyring.Keyring }

func New(appName string) (Store, error) {
	kr, err := keyring.Open(keyring.Config{
		ServiceName:              appName,
		KeychainName:             appName,
		KeychainTrustApplication: true,
		// Defaults work cross-platform; refine in later phases.
	})
	if err != nil {
		return nil, err
	}
	return &store{kr: kr}, nil
}

func (s *store) Get(key string) (keyring.Item, error) { return s.kr.Get(key) }
func (s *store) Set(item keyring.Item) error          { return s.kr.Set(item) }
func (s *store) Delete(key string) error              { return s.kr.Remove(key) }
