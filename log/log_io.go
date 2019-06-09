package log

import (
	"time"

	"github.com/berty/go-ipfs-log/entry"
	"github.com/berty/go-ipfs-log/errmsg"
	"github.com/berty/go-ipfs-log/io"
	"github.com/berty/go-ipfs-log/utils/lamportclock"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/pkg/errors"
)

type FetchOptions struct {
	Length       *int
	Exclude      []*entry.Entry
	ProgressChan chan *entry.Entry
	Timeout      time.Duration
}

func ToMultihash(services *io.IpfsServices, log *Log) (cid.Cid, error) {
	if log.Values().Len() < 1 {
		return cid.Cid{}, errors.New(`Can't serialize an empty log`)
	}

	return io.WriteCBOR(services, log.ToJSON())
}

func FromMultihash(services *io.IpfsServices, hash cid.Cid, options *FetchOptions) (*Snapshot, error) {
	result, err := io.ReadCBOR(services, hash)
	if err != nil {
		return nil, err
	}

	logData := &JSONLog{}
	err = cbornode.DecodeInto(result.RawData(), logData)
	if err != nil {
		return nil, err
	}

	entries := entry.FetchAll(services, logData.Heads, &entry.FetchOptions{
		Length:       options.Length,
		Exclude:      options.Exclude,
		ProgressChan: options.ProgressChan,
	})

	// Find latest clock
	var clock *lamportclock.LamportClock
	for _, e := range entries {
		if clock == nil || e.Clock.Time > clock.Time {
			clock = lamportclock.New(e.Clock.ID, e.Clock.Time)
		}
	}

	entry.SortEntries(entries)

	heads := []*entry.Entry{}
	for _, e := range entries {
		for _, h := range logData.Heads {
			if h.String() == e.Hash.String() {
				heads = append(heads, e)
			}
		}
	}

	headsCids := []cid.Cid{}
	for _, head := range heads {
		headsCids = append(headsCids, head.Hash)
	}

	return &Snapshot{
		ID:     logData.ID,
		Values: entries,
		Heads:  headsCids,
		Clock:  clock,
	}, nil
}

func FromEntryHash(services *io.IpfsServices, hashes []cid.Cid, options *FetchOptions) ([]*entry.Entry, error) {
	if services == nil {
		return nil, errmsg.IPFSNotDefined
	}

	if options == nil {
		return nil, errmsg.FetchOptionsNotDefined
	}

	// Fetch given length, return size at least the given input entries
	length := -1
	if options.Length != nil && *options.Length > -1 {
		length = maxInt(*options.Length, 1)
	}

	entries := entry.FetchParallel(services, hashes, &entry.FetchOptions{
		Length:       options.Length,
		Exclude:      options.Exclude,
		ProgressChan: options.ProgressChan,
	})

	sliced := entries
	if length > -1 {
		entries = entries[:-length]
	}

	return sliced, nil
}

func FromJSON(services *io.IpfsServices, jsonLog *JSONLog, options *entry.FetchOptions) (*Snapshot, error) {
	if services == nil {
		return nil, errmsg.IPFSNotDefined
	}

	if options == nil {
		return nil, errmsg.FetchOptionsNotDefined
	}

	entries := entry.FetchParallel(services, jsonLog.Heads, &entry.FetchOptions{
		Length:       options.Length,
		Exclude:      []*entry.Entry{},
		ProgressChan: options.ProgressChan,
		Concurrency:  16,
		Timeout:      options.Timeout,
	})

	entry.SortEntries(entries)

	return &Snapshot{
		ID:     jsonLog.ID,
		Heads:  jsonLog.Heads,
		Values: entries,
	}, nil
}

func FromEntry(services *io.IpfsServices, sourceEntries []*entry.Entry, options *entry.FetchOptions) (*Snapshot, error) {
	if services == nil {
		return nil, errmsg.IPFSNotDefined
	}

	if options == nil {
		return nil, errmsg.FetchOptionsNotDefined
	}

	// Fetch given length, return size at least the given input entries
	length := -1
	if options.Length != nil && *options.Length > -1 {
		length = maxInt(*options.Length, len(sourceEntries))
	}

	// Make sure we pass hashes instead of objects to the fetcher function
	hashes := []cid.Cid{}
	for _, e := range sourceEntries {
		hashes = append(hashes, e.Hash)
	}

	// Fetch the entries
	entries := entry.FetchParallel(services, hashes, &entry.FetchOptions{
		Length:       &length,
		Exclude:      options.Exclude,
		ProgressChan: options.ProgressChan,
	})

	// Combine the fetches with the source entries and take only uniques
	combined := append(sourceEntries, entries...)
	uniques := entry.NewOrderedMapFromEntries(combined).Slice()
	entry.SortEntries(uniques)

	// Cap the result at the right size by taking the last n entries
	var sliced []*entry.Entry

	if length > -1 {
		sliced = entry.Slice(uniques, -length)
	} else {
		sliced = uniques
	}

	missingSourceEntries := entry.Difference(sliced, sourceEntries)
	result := append(missingSourceEntries, entry.SliceRange(sliced, len(missingSourceEntries), len(sliced))...)

	return &Snapshot{
		ID:     result[len(result)-1].LogID,
		Values: result,
	}, nil
}
