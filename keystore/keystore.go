package keystore

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/pkg/errors"
)

type Keystore struct {
	store datastore.Datastore
	cache *lru.Cache
}

func (k *Keystore) Sign(pubKey crypto.PrivKey, bytes []byte) ([]byte, error) {
	return pubKey.Sign(bytes)
}

func (k *Keystore) Verify(signature []byte, publicKey crypto.PubKey, data []byte) error {
	ok, err := publicKey.Verify(data, signature)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("signature is not valid for the supplied data")
	}

	return nil
}

func NewKeystore(store datastore.Datastore) (*Keystore, error) {
	cache, err := lru.New(128)
	if err != nil {
		return nil, err
	}

	return &Keystore{
		store: store,
		cache: cache,
	}, nil
}

func (k *Keystore) HasKey(id string) (bool, error) {
	hasKey := false
	storedKey, ok := k.cache.Peek(id)

	if ok == false {
		value, err := k.store.Get(datastore.NewKey(id))
		if err != nil {
			return false, err
		}

		if storedKey != nil {
			k.cache.Add(id, base64.StdEncoding.EncodeToString(value))
		}
	}

	hasKey = storedKey != nil

	return hasKey, nil
}

func (k *Keystore) CreateKey(id string) (crypto.PrivKey, error) {
	// FIXME: I kept Secp256k1 for compatibility with OrbitDB, should we change this?
	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	keyBytes, err := priv.Raw()
	if err != nil {
		return nil, err
	}

	if err := k.store.Put(datastore.NewKey(id), keyBytes); err != nil {
		return nil, err
	}

	k.cache.Add(id, base64.StdEncoding.EncodeToString(keyBytes))

	return priv, nil
}

func (k *Keystore) GetKey(id string) (crypto.PrivKey, error) {
	var err error
	var keyBytes []byte

	cachedKey, ok := k.cache.Get(id)
	if !ok || cachedKey == nil {
		keyBytes, err = k.store.Get(datastore.NewKey(id))

		if err != nil {
			cachedKey = nil
			return nil, errors.New("unable to fetch a private key from keystore")
		} else {
			k.cache.Add(id, base64.StdEncoding.EncodeToString(keyBytes))
		}
	} else {
		keyBytes, err = base64.StdEncoding.DecodeString(cachedKey.(string))
		if err != nil {
			return nil, errors.Wrap(err,"unable to cast private key to bytes")
		}
	}

	privateKey, err := crypto.UnmarshalSecp256k1PrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

var _ Interface = &Keystore{}
