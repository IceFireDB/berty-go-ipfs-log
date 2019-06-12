package example

import (
	"context"
	"fmt"
	"io/ioutil"

	idp "berty.tech/go-ipfs-log/identityprovider"
	log_io "berty.tech/go-ipfs-log/io"
	keystore "berty.tech/go-ipfs-log/keystore"
	"berty.tech/go-ipfs-log/log"
	datastore "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	config "github.com/ipfs/go-ipfs-config"
	ipfs_core "github.com/ipfs/go-ipfs/core"
	ipfs_libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	ipfs_repo "github.com/ipfs/go-ipfs/repo"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

func buildHostOverrideExample(ctx context.Context, id peer.ID, ps pstore.Peerstore, options ...libp2p.Option) (host.Host, error) {
	return ipfs_libp2p.DefaultHostOption(ctx, id, ps, options...)
}

func newRepo() (ipfs_repo.Repo, error) {
	// Generating config
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return nil, err
	}

	// Listen on local interface only
	cfg.Addresses.Swarm = []string{
		"/ip4/127.0.0.1/tcp/0",
	}

	// Do not bootstrap on ipfs node
	cfg.Bootstrap = []string{}

	return &ipfs_repo.Mock{
		D: dssync.MutexWrap(datastore.NewMapDatastore()),
		C: *cfg,
	}, nil
}

func buildNode(ctx context.Context) (*ipfs_core.IpfsNode, error) {
	r, err := newRepo()
	if err != nil {
		return nil, err
	}

	cfg := &ipfs_core.BuildCfg{
		Online: true,
		Repo:   r,
		Host:   buildHostOverrideExample,
	}

	return ipfs_core.NewNode(ctx, cfg)
}

func ExampleLogAppend() {
	ctx := context.Background()

	// Build Ipfs Node A
	nodeA, err := buildNode(ctx)
	if err != nil {
		panic(err)
	}

	// Build Ipfs Node B
	nodeB, err := buildNode(ctx)
	if err != nil {
		panic(err)
	}

	nodeBInfo := pstore.PeerInfo{
		ID:    nodeB.Identity,
		Addrs: nodeB.PeerHost.Addrs(),
	}

	// Connecting NodeA with NodeB
	if err := nodeA.PeerHost.Connect(ctx, nodeBInfo); err != nil {
		panic(fmt.Errorf("connect error: %s", err))
	}

	mdsA := datastore.NewMapDatastore()
	serviceA := log_io.FromIpfsNode(nodeA, mdsA)

	mdsB := datastore.NewMapDatastore()
	serviceB := log_io.FromIpfsNode(nodeB, mdsB)

	// Fill up datastore with identities
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	ks, err := keystore.NewKeystore(ds)
	if err != nil {
		panic(err)
	}

	// Create identity A
	identityA, err := idp.CreateIdentity(&idp.CreateIdentityOptions{
		Keystore: ks,
		ID:       "userA",
		Type:     "orbitdb",
	})

	if err != nil {
		panic(err)
	}

	// Create identity B
	identityB, err := idp.CreateIdentity(&idp.CreateIdentityOptions{
		Keystore: ks,
		ID:       "userB",
		Type:     "orbitdb",
	})

	if err != nil {
		panic(err)
	}

	// creating log
	logA, err := log.NewLog(serviceA, identityA, &log.NewLogOptions{ID: "A"})
	if err != nil {
		panic(err)
	}

	// nodeA Append data (hello world)"
	_, err = logA.Append([]byte("hello world"), 1)
	if err != nil {
		panic(fmt.Errorf("append error: %s", err))
	}

	h, err := logA.ToMultihash()
	if err != nil {
		panic(fmt.Errorf("ToMultihash error: %s", err))
	}

	res, err := log.NewFromMultihash(serviceB, identityB, h, &log.NewLogOptions{}, &log.FetchOptions{})
	if err != nil {
		panic(fmt.Errorf("NewFromMultihash error: %s", err))
	}

	// nodeB lookup logA
	fmt.Println(res.ToString(nil))

	// Output: hello world
}
