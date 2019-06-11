package test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/berty/go-ipfs-log/entry"
	idp "github.com/berty/go-ipfs-log/identityprovider"
	"github.com/berty/go-ipfs-log/io"
	ks "github.com/berty/go-ipfs-log/keystore"
	"github.com/berty/go-ipfs-log/log"
	"github.com/berty/go-ipfs-log/test/logcreator"
	"github.com/ipfs/go-cid"
	_ "github.com/ipfs/go-cid"
	dssync "github.com/ipfs/go-datastore/sync"
	"math/rand"
	_ "sort"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func BadComparatorReturnsZero (a *entry.Entry, b *entry.Entry) (int, error) {
	return 0, nil
}

func TestLogLoad(t *testing.T) {
	_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	ipfs := io.NewMemoryServices()

	datastore := dssync.MutexWrap(NewIdentityDataStore())
	keystore, err := ks.NewKeystore(datastore)
	if err != nil {
		panic(err)
	}

	var identities [4]*idp.Identity

	for i, char := range []rune{'C', 'B', 'D', 'A'} {
		identity, err := idp.CreateIdentity(&idp.CreateIdentityOptions{
			Keystore: keystore,
			ID:       fmt.Sprintf("user%c", char),
			Type:     "orbitdb",
		})

		if err != nil {
			panic(err)
		}

		identities[i] = identity
	}

	firstWriteExpectedData := []string{
		"entryA6", "entryA7", "entryA8", "entryA9",
		"entryA10", "entryB1", "entryB2", "entryB3",
		"entryB4", "entryB5", "entryA1", "entryA2",
		"entryA3", "entryA4", "entryA5", "entryC0",
	}

	_ = firstWriteExpectedData

	Convey("Log - Load", t, FailureHalts, func(c C) {
		c.Convey("fromJSON", FailureHalts, func(c C) {
			c.Convey("creates a log from an entry", FailureHalts, func(c C) {
				fixture, err := logcreator.CreateLogWithSixteenEntries(ipfs, identities)
				c.So(err, ShouldBeNil)

				data := fixture.Log
				json := fixture.JSON

				// TODO: Is this useless?
				//heads := map[string]*entry.Entry{}
				//
				//for _, h := range json.Heads {
				//	e, err := entry.FromMultihash(ipfs, h, identities[0].Provider)
				//	c.So(err, ShouldBeNil)
				//
				//	heads[e.Hash.String()] = e
				//}

				l, err := log.NewFromJSON(ipfs, identities[0], json, &log.NewLogOptions{ID: "X"}, &entry.FetchOptions{})
				c.So(err, ShouldBeNil)

				values := l.Values()

				c.So(l.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(values.Len(), ShouldEqual, 16)
				c.So(entriesAsStrings(values), ShouldResemble, fixture.ExpectedData)
			})

			c.Convey("creates a log from an entry with custom tiebreaker", FailureHalts, func(c C) {
				fixture, err := logcreator.CreateLogWithSixteenEntries(ipfs, identities)
				c.So(err, ShouldBeNil)

				data := fixture.Log
				json := fixture.JSON

				l, err := log.NewFromJSON(ipfs, identities[0], json, &log.NewLogOptions{ID: "X", SortFn: log.FirstWriteWins}, &entry.FetchOptions{Length: intPtr(-1)})
				c.So(err, ShouldBeNil)

				c.So(l.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(l.Values().Len(), ShouldEqual, 16)
				// TODO: found out why firstWriteExpectedData is what it is in JS test

				c.So(entriesAsStrings(l.Values()), ShouldResemble, firstWriteExpectedData)
				_ = firstWriteExpectedData
			})
		})

		c.Convey("fromEntryHash", FailureHalts, func(c C) {
			c.Convey("creates a log from an entry hash", FailureHalts, func(c C) {
				fixture, err := logcreator.CreateLogWithSixteenEntries(ipfs, identities)
				c.So(err, ShouldBeNil)

				data := fixture.Log
				json := fixture.JSON

				log1, err := log.NewFromEntryHash(ipfs, identities[0], json.Heads[0], &log.NewLogOptions{ID: "X"}, &log.FetchOptions{})
				log2, err := log.NewFromEntryHash(ipfs, identities[0], json.Heads[1], &log.NewLogOptions{ID: "X"}, &log.FetchOptions{})

				_, err = log1.Join(log2, -1)
				c.So(err, ShouldBeNil)

				c.So(log1.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(log1.Values().Len(), ShouldEqual, 16)
				c.So(entriesAsStrings(log1.Values()), ShouldResemble, fixture.ExpectedData)
			})

			c.Convey("creates a log from an entry hash with custom tiebreaker", FailureHalts, func(c C) {
				fixture, err := logcreator.CreateLogWithSixteenEntries(ipfs, identities)
				c.So(err, ShouldBeNil)

				data := fixture.Log
				json := fixture.JSON

				log1, err := log.NewFromEntryHash(ipfs, identities[0], json.Heads[0], &log.NewLogOptions{ID: "X", SortFn: log.FirstWriteWins}, &log.FetchOptions{})
				log2, err := log.NewFromEntryHash(ipfs, identities[0], json.Heads[1], &log.NewLogOptions{ID: "X", SortFn: log.FirstWriteWins}, &log.FetchOptions{})

				_, err = log1.Join(log2, -1)
				c.So(err, ShouldBeNil)

				c.So(log1.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(log1.Values().Len(), ShouldEqual, 16)
				c.So(entriesAsStrings(log1.Values()), ShouldResemble, firstWriteExpectedData)
			})
		})

		c.Convey("fromEntry", FailureHalts, func(c C) {
			resortedIdentities := [4]*idp.Identity{identities[2], identities[1], identities[0], identities[3]}

			c.Convey("creates a log from an entry", FailureHalts, func(c C) {
				fixture, err := logcreator.CreateLogWithSixteenEntries(ipfs, resortedIdentities)
				c.So(err, ShouldBeNil)

				data := fixture.Log

				l, err := log.NewFromEntry(ipfs, identities[0], data.Heads().Slice(), &log.NewLogOptions{}, &entry.FetchOptions{})
				c.So(err, ShouldBeNil)

				c.So(l.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(l.Values().Len(), ShouldEqual, 16)
				c.So(entriesAsStrings(l.Values()), ShouldResemble, fixture.ExpectedData)
			})

			c.Convey("creates a log from an entry with custom tiebreaker", FailureHalts, func(c C) {
				fixture, err := logcreator.CreateLogWithSixteenEntries(ipfs, resortedIdentities)
				c.So(err, ShouldBeNil)

				data := fixture.Log

				l, err := log.NewFromEntry(ipfs, identities[0], data.Heads().Slice(), &log.NewLogOptions{SortFn: log.FirstWriteWins}, &entry.FetchOptions{Length: intPtr(-1)})
				c.So(err, ShouldBeNil)

				c.So(l.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(l.Values().Len(), ShouldEqual, 16)
				c.So(entriesAsStrings(l.Values()), ShouldResemble, firstWriteExpectedData)
			})

			c.Convey("keeps the original heads", FailureHalts, func(c C) {
				fixture, err := logcreator.CreateLogWithSixteenEntries(ipfs, resortedIdentities)
				c.So(err, ShouldBeNil)

				data := fixture.Log

				log1, err := log.NewFromEntry(ipfs, identities[0], data.Heads().Slice(), &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(data.Heads().Len())})

				c.So(err, ShouldBeNil)
				c.So(log1.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(log1.Values().Len(), ShouldEqual, data.Heads().Len())
				c.So(string(log1.Values().At(0).Payload), ShouldEqual, "entryC0")
				c.So(string(log1.Values().At(1).Payload), ShouldEqual, "entryA10")

				log2, err := log.NewFromEntry(ipfs, identities[0], data.Heads().Slice(), &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(4)})

				c.So(err, ShouldBeNil)
				c.So(log2.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(log2.Values().Len(), ShouldEqual, 4)
				c.So(string(log2.Values().At(0).Payload), ShouldEqual, "entryC0")
				c.So(string(log2.Values().At(1).Payload), ShouldEqual, "entryA8")
				c.So(string(log2.Values().At(2).Payload), ShouldEqual, "entryA9")
				c.So(string(log2.Values().At(3).Payload), ShouldEqual, "entryA10")

				log3, err := log.NewFromEntry(ipfs, identities[0], data.Heads().Slice(), &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(7)})

				c.So(err, ShouldBeNil)
				c.So(log3.ID, ShouldEqual, data.Heads().At(0).LogID)
				c.So(log3.Values().Len(), ShouldEqual, 7)
				c.So(string(log3.Values().At(0).Payload), ShouldEqual, "entryB5")
				c.So(string(log3.Values().At(1).Payload), ShouldEqual, "entryA6")
				c.So(string(log3.Values().At(2).Payload), ShouldEqual, "entryC0")
				c.So(string(log3.Values().At(3).Payload), ShouldEqual, "entryA7")
				c.So(string(log3.Values().At(4).Payload), ShouldEqual, "entryA8")
				c.So(string(log3.Values().At(5).Payload), ShouldEqual, "entryA9")
				c.So(string(log3.Values().At(6).Payload), ShouldEqual, "entryA10")
			})

			c.Convey("onProgress callback is fired for each entry", FailureHalts, func(c C) {
				// TODO: skipped
			})

			c.Convey("retrieves partial log from an entry hash", FailureHalts, func(c C) {
				log1, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log2, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log3, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				var items1 []*entry.Entry
				var items2 []*entry.Entry
				var items3 []*entry.Entry

				const amount = 100
				for i := 1; i <= amount; i++ {
					prev1 := lastEntry(items1)
					prev2 := lastEntry(items2)
					prev3 := lastEntry(items3)

					var nexts []cid.Cid
					if prev1 != nil {
						nexts = []cid.Cid{prev1.Hash}
					}

					n1, err := entry.CreateEntry(ipfs, log1.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryA%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					if prev2 != nil {
						nexts = []cid.Cid{prev2.Hash, n1.Hash}
					} else {
						nexts = []cid.Cid{n1.Hash}
					}

					n2, err := entry.CreateEntry(ipfs, log2.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryB%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					if prev3 != nil {
						nexts = []cid.Cid{prev3.Hash, n1.Hash, n2.Hash}
					} else {
						nexts = []cid.Cid{n1.Hash, n2.Hash}
					}

					n3, err := entry.CreateEntry(ipfs, log3.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryC%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					items1 = append(items1, n1)
					items2 = append(items2, n2)
					items3 = append(items3, n3)
				}

				// limit to 10 entries
				a, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(10)})
				c.So(err, ShouldBeNil)
				c.So(a.Values().Len(), ShouldEqual, 10)

				// limit to 42 entries
				b, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(42)})
				c.So(err, ShouldBeNil)
				c.So(b.Values().Len(), ShouldEqual, 42)
			})

			c.Convey("retrieves full log from an entry hash", FailureHalts, func(c C) {
				log1, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log2, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log3, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				var items1 []*entry.Entry
				var items2 []*entry.Entry
				var items3 []*entry.Entry

				const amount = 100
				for i := 1; i <= amount; i++ {
					prev1 := lastEntry(items1)
					prev2 := lastEntry(items2)
					prev3 := lastEntry(items3)

					var nexts []cid.Cid
					if prev1 != nil {
						nexts = []cid.Cid{prev1.Hash}
					}

					n1, err := entry.CreateEntry(ipfs, log1.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryA%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					if prev2 != nil {
						nexts = []cid.Cid{prev2.Hash, n1.Hash}
					} else {
						nexts = []cid.Cid{n1.Hash}
					}

					n2, err := entry.CreateEntry(ipfs, log2.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryB%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					if prev3 != nil {
						nexts = []cid.Cid{prev3.Hash, n2.Hash}
					} else {
						nexts = []cid.Cid{n2.Hash}
					}

					n3, err := entry.CreateEntry(ipfs, log3.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryC%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					items1 = append(items1, n1)
					items2 = append(items2, n2)
					items3 = append(items3, n3)
				}

				lA, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 1)})
				c.So(err, ShouldBeNil)
				c.So(lA.Values().Len(), ShouldEqual, amount)

				lB, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items2)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 2)})
				c.So(err, ShouldBeNil)
				c.So(lB.Values().Len(), ShouldEqual, amount*2)

				lC, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items3)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 3)})
				c.So(err, ShouldBeNil)
				c.So(lC.Values().Len(), ShouldEqual, amount*3)
			})

			c.Convey("retrieves full log from an entry hash 2", FailureHalts, func(c C) {
				log1, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log2, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log3, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				var items1 []*entry.Entry
				var items2 []*entry.Entry
				var items3 []*entry.Entry

				const amount = 100
				for i := 1; i <= amount; i++ {
					prev1 := lastEntry(items1)
					prev2 := lastEntry(items2)
					prev3 := lastEntry(items3)

					var nexts []cid.Cid
					if prev1 != nil {
						nexts = []cid.Cid{prev1.Hash}
					}

					n1, err := entry.CreateEntry(ipfs, log1.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryA%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					if prev2 != nil {
						nexts = []cid.Cid{prev2.Hash, n1.Hash}
					} else {
						nexts = []cid.Cid{n1.Hash}
					}

					n2, err := entry.CreateEntry(ipfs, log2.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryB%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					if prev3 != nil {
						nexts = []cid.Cid{prev3.Hash, n1.Hash, n2.Hash}
					} else {
						nexts = []cid.Cid{n1.Hash, n2.Hash}
					}

					n3, err := entry.CreateEntry(ipfs, log3.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryC%d", i)), Next: nexts}, nil)
					c.So(err, ShouldBeNil)

					items1 = append(items1, n1)
					items2 = append(items2, n2)
					items3 = append(items3, n3)
				}

				lA, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 1)})
				c.So(err, ShouldBeNil)
				c.So(lA.Values().Len(), ShouldEqual, amount)

				lB, err := log.NewFromEntry(ipfs, identities[1], []*entry.Entry{lastEntry(items2)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 2)})
				c.So(err, ShouldBeNil)
				c.So(lB.Values().Len(), ShouldEqual, amount*2)

				lC, err := log.NewFromEntry(ipfs, identities[2], []*entry.Entry{lastEntry(items3)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 3)})
				c.So(err, ShouldBeNil)
				c.So(lC.Values().Len(), ShouldEqual, amount*3)
			})

			c.Convey("retrieves full log from an entry hash 3", FailureHalts, func(c C) {
				log1, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log2, err := log.NewLog(ipfs, identities[1], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log3, err := log.NewLog(ipfs, identities[3], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				var items1 []*entry.Entry
				var items2 []*entry.Entry
				var items3 []*entry.Entry

				const amount = 10
				for i := 1; i <= amount; i++ {
					prev1 := lastEntry(items1)
					prev2 := lastEntry(items2)
					prev3 := lastEntry(items3)

					var nexts []cid.Cid

					log1.Clock.Tick()
					log2.Clock.Tick()
					log3.Clock.Tick()

					if prev1 != nil {
						nexts = []cid.Cid{prev1.Hash}
					}

					n1, err := entry.CreateEntry(ipfs, log1.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryA%d", i)), Next: nexts}, log1.Clock)
					c.So(err, ShouldBeNil)

					if prev2 != nil {
						nexts = []cid.Cid{prev2.Hash, n1.Hash}
					} else {
						nexts = []cid.Cid{n1.Hash}
					}

					n2, err := entry.CreateEntry(ipfs, log2.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryB%d", i)), Next: nexts}, log2.Clock)
					c.So(err, ShouldBeNil)

					if prev3 != nil {
						nexts = []cid.Cid{prev3.Hash, n1.Hash, n2.Hash}
					} else {
						nexts = []cid.Cid{n1.Hash, n2.Hash}
					}

					n3, err := entry.CreateEntry(ipfs, log3.Identity, &entry.Entry{LogID: "X", Payload: []byte(fmt.Sprintf("entryC%d", i)), Next: nexts}, log3.Clock)
					c.So(err, ShouldBeNil)

					log1.Clock.Merge(log2.Clock)
					log1.Clock.Merge(log3.Clock)
					log2.Clock.Merge(log1.Clock)
					log2.Clock.Merge(log3.Clock)
					log3.Clock.Merge(log1.Clock)
					log3.Clock.Merge(log2.Clock)

					items1 = append(items1, n1)
					items2 = append(items2, n2)
					items3 = append(items3, n3)
				}

				lA, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 1)})
				c.So(err, ShouldBeNil)
				c.So(lA.Values().Len(), ShouldEqual, amount)

				itemsInB := []string{
					"entryA1",
					"entryB1",
					"entryA2",
					"entryB2",
					"entryA3",
					"entryB3",
					"entryA4",
					"entryB4",
					"entryA5",
					"entryB5",
					"entryA6",
					"entryB6",
					"entryA7",
					"entryB7",
					"entryA8",
					"entryB8",
					"entryA9",
					"entryB9",
					"entryA10",
					"entryB10",
				}

				lB, err := log.NewFromEntry(ipfs, identities[1], []*entry.Entry{lastEntry(items2)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 2)})
				c.So(err, ShouldBeNil)
				c.So(lB.Values().Len(), ShouldEqual, amount*2)
				c.So(entriesAsStrings(lB.Values()), ShouldResemble, itemsInB)

				lC, err := log.NewFromEntry(ipfs, identities[3], []*entry.Entry{lastEntry(items3)}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(amount * 3)})
				c.So(err, ShouldBeNil)

				_, err = lC.Append([]byte("EOF"), 1)
				c.So(err, ShouldBeNil)

				c.So(lC.Values().Len(), ShouldEqual, amount*3+1)

				tmp := []string{
					"entryA1",
					"entryB1",
					"entryC1",
					"entryA2",
					"entryB2",
					"entryC2",
					"entryA3",
					"entryB3",
					"entryC3",
					"entryA4",
					"entryB4",
					"entryC4",
					"entryA5",
					"entryB5",
					"entryC5",
					"entryA6",
					"entryB6",
					"entryC6",
					"entryA7",
					"entryB7",
					"entryC7",
					"entryA8",
					"entryB8",
					"entryC8",
					"entryA9",
					"entryB9",
					"entryC9",
					"entryA10",
					"entryB10",
					"entryC10",
					"EOF",
				}

				c.So(entriesAsStrings(lC.Values()), ShouldResemble, tmp)

				// make sure logX comes after A, B and C
				logX, err := log.NewLog(ipfs, identities[3], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				_, err = logX.Append([]byte{'1'}, 1)
				c.So(err, ShouldBeNil)

				_, err = logX.Append([]byte{'2'}, 1)
				c.So(err, ShouldBeNil)

				_, err = logX.Append([]byte{'3'}, 1)
				c.So(err, ShouldBeNil)

				lD, err := log.NewFromEntry(ipfs, identities[2], []*entry.Entry{lastEntry(logX.Values().Slice())}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(-1)})
				c.So(err, ShouldBeNil)

				_, err = lC.Join(lD, -1)
				c.So(err, ShouldBeNil)

				_, err = lD.Join(lC, -1)
				c.So(err, ShouldBeNil)

				_, err = lC.Append([]byte("DONE"), 1)
				c.So(err, ShouldBeNil)

				_, err = lD.Append([]byte("DONE"), 1)
				c.So(err, ShouldBeNil)

				logF, err := log.NewFromEntry(ipfs, identities[2], []*entry.Entry{lastEntry(lC.Values().Slice())}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(-1), Exclude: nil})
				c.So(err, ShouldBeNil)

				logG, err := log.NewFromEntry(ipfs, identities[2], []*entry.Entry{lastEntry(lD.Values().Slice())}, &log.NewLogOptions{}, &entry.FetchOptions{Length: intPtr(-1), Exclude: nil})
				c.So(err, ShouldBeNil)

				c.So(logF.ToString(nil), ShouldEqual, bigLogString)
				c.So(logG.ToString(nil), ShouldEqual, bigLogString)
			})

			c.Convey("retrieves full log of randomly joined log", FailureHalts, func(c C) {
				log1, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log2, err := log.NewLog(ipfs, identities[1], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log3, err := log.NewLog(ipfs, identities[3], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				for i := 1; i <= 5; i++ {
					_, err := log1.Append([]byte(fmt.Sprintf("entryA%d", i)), 1)
					c.So(err, ShouldBeNil)

					_, err = log2.Append([]byte(fmt.Sprintf("entryB%d", i)), 1)
					c.So(err, ShouldBeNil)
				}

				_, err = log3.Join(log1, -1)
				c.So(err, ShouldBeNil)

				_, err = log3.Join(log2, -1)
				c.So(err, ShouldBeNil)

				for i := 6; i <= 10; i++ {
					_, err := log1.Append([]byte(fmt.Sprintf("entryA%d", i)), 1)
					c.So(err, ShouldBeNil)
				}

				_, err = log1.Join(log3, -1)

				for i := 11; i <= 15; i++ {
					_, err := log1.Append([]byte(fmt.Sprintf("entryA%d", i)), 1)
					c.So(err, ShouldBeNil)
				}

				expectedData := []string{"entryA1", "entryB1", "entryA2", "entryB2",
					"entryA3", "entryB3", "entryA4", "entryB4",
					"entryA5", "entryB5",
					"entryA6", "entryA7", "entryA8", "entryA9", "entryA10",
					"entryA11", "entryA12", "entryA13", "entryA14", "entryA15",
				}

				c.So(entriesAsStrings(log1.Values()), ShouldResemble, expectedData)
			})

			c.Convey("retrieves randomly joined log deterministically", FailureHalts, func(c C) {
				logA, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				logB, err := log.NewLog(ipfs, identities[2], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				log3, err := log.NewLog(ipfs, identities[3], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				l, err := log.NewLog(ipfs, identities[1], &log.NewLogOptions{ID: "X"})
				c.So(err, ShouldBeNil)

				for i := 1; i <= 5; i++ {
					_, err := logA.Append([]byte(fmt.Sprintf("entryA%d", i)), 1)
					c.So(err, ShouldBeNil)

					_, err = logB.Append([]byte(fmt.Sprintf("entryB%d", i)), 1)
					c.So(err, ShouldBeNil)
				}

				_, err = log3.Join(logA, -1)
				c.So(err, ShouldBeNil)

				_, err = log3.Join(logB, -1)
				c.So(err, ShouldBeNil)

				for i := 6; i <= 10; i++ {
					_, err := logA.Append([]byte(fmt.Sprintf("entryA%d", i)), 1)
					c.So(err, ShouldBeNil)
				}

				_, err = l.Join(log3, -1)
				c.So(err, ShouldBeNil)

				_, err = l.Append([]byte("entryC0"), 1)
				c.So(err, ShouldBeNil)

				_, err = l.Join(logA, 16)
				c.So(err, ShouldBeNil)

				expectedData := []string{
					"entryA1", "entryB1", "entryA2", "entryB2",
					"entryA3", "entryB3", "entryA4", "entryB4",
					"entryA5", "entryB5",
					"entryA6",
					"entryC0", "entryA7", "entryA8", "entryA9", "entryA10",
				}

				c.So(entriesAsStrings(l.Values()), ShouldResemble, expectedData)
			})

			c.Convey("sorts", FailureHalts, func(c C) {
				testLog, err := logcreator.CreateLogWithSixteenEntries(ipfs, resortedIdentities)
				c.So(err, ShouldBeNil)

				l := testLog.Log
				expectedData := testLog.ExpectedData

				expectedData2 := []string{
					"entryA1", "entryB1", "entryA2", "entryB2",
					"entryA3", "entryB3", "entryA4", "entryB4",
					"entryA5", "entryB5",
					"entryA6", "entryA7", "entryA8", "entryA9", "entryA10",
				}

				expectedData3 := []string{
					"entryA1", "entryB1", "entryA2", "entryB2",
					"entryA3", "entryB3", "entryA4", "entryB4",
					"entryA5", "entryB5", "entryA6", "entryC0",
					"entryA7", "entryA8", "entryA9",
				}

				expectedData4 := []string{
					"entryA1", "entryB1", "entryA2", "entryB2",
					"entryA3", "entryB3", "entryA4", "entryB4",
					"entryA5", "entryA6", "entryC0", "entryA7",
					"entryA8", "entryA9", "entryA10",
				}

				fetchOrder := l.Values().Slice()
				entry.Sort(entry.Compare, fetchOrder)
				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(fetchOrder)), ShouldResemble, expectedData)

				reverseOrder := l.Values().Slice()
				log.Reverse(reverseOrder)
				entry.Sort(entry.Compare, reverseOrder)
				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(reverseOrder)), ShouldResemble, expectedData)

				hashOrder := l.Values().Slice()
				entry.Sort(func (a, b *entry.Entry) (int, error) {
					return strings.Compare(a.Hash.String(), b.Hash.String()), nil
				}, hashOrder)
				entry.Sort(entry.Compare, hashOrder)
				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(hashOrder)), ShouldResemble, expectedData)

				var partialLog []*entry.Entry
				for _, item := range l.Values().Slice() {
					if bytes.Compare(item.Payload, []byte("entryC0")) != 0 {
						partialLog = append(partialLog, item)
					}
				}
				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(partialLog)), ShouldResemble, expectedData2)

				var partialLog2 []*entry.Entry
				for _, item := range l.Values().Slice() {
					if bytes.Compare(item.Payload, []byte("entryA10")) != 0 {
						partialLog2 = append(partialLog2, item)
					}
				}
				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(partialLog2)), ShouldResemble, expectedData3)

				var partialLog3 []*entry.Entry
				for _, item := range l.Values().Slice() {
					if bytes.Compare(item.Payload, []byte("entryB5")) != 0 {
						partialLog3 = append(partialLog3, item)
					}
				}
				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(partialLog3)), ShouldResemble, expectedData4)
			})

			c.Convey("sorts deterministically from random order", FailureHalts, func(c C) {
				testLog, err := logcreator.CreateLogWithSixteenEntries(ipfs, resortedIdentities)
				c.So(err, ShouldBeNil)

				l := testLog.Log
				expectedData := testLog.ExpectedData

				fetchOrder := l.Values().Slice()
				entry.Sort(entry.Compare, fetchOrder)
				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(fetchOrder)), ShouldResemble, expectedData)

				for i := 0; i < 1000; i++ {
					randomOrder := l.Values().Slice()
					entry.Sort(func (a, b *entry.Entry) (int, error) {
						return rand.Int(), nil
					}, randomOrder)
					entry.Sort(entry.Compare, randomOrder)

					c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(randomOrder)), ShouldResemble, expectedData)
				}
			})

			c.Convey("sorts entries correctly", FailureHalts, func(c C) {
				testLog, err := logcreator.CreateLogWithHundredEntries(ipfs, resortedIdentities)
				c.So(err, ShouldBeNil)

				l := testLog.Log
				expectedData := testLog.ExpectedData

				c.So(entriesAsStrings(entry.NewOrderedMapFromEntries(l.Values().Slice())), ShouldResemble, expectedData)
			})

			c.Convey("sorts entries according to custom tiebreaker function", FailureHalts, func(c C) {
				testLog, err := logcreator.CreateLogWithSixteenEntries(ipfs, resortedIdentities)
				c.So(err, ShouldBeNil)

				firstWriteWinsLog, err := log.NewLog(ipfs, resortedIdentities[0], &log.NewLogOptions{ ID: "X", SortFn: BadComparatorReturnsZero })
				c.So(err, ShouldBeNil)

				_, err = firstWriteWinsLog.Join(testLog.Log, -1)
				// TODO: the error is only thrown silently when calling .Values(), should we handle it properly
				//firstWriteWinsLog.Values()
				//c.So(err, ShouldNotBeNil)
			})

			c.Convey("retrieves partially joined log deterministically - single next pointer", FailureHalts, func(c C) {
				nextPointersAmount := 1

				logA, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ ID:"X" })
				c.So(err, ShouldBeNil)
				logB, err := log.NewLog(ipfs, identities[2], &log.NewLogOptions{ ID:"X" })
				c.So(err, ShouldBeNil)
				log3, err := log.NewLog(ipfs, identities[3], &log.NewLogOptions{ ID: "X" })
				c.So(err, ShouldBeNil)
				l, err := log.NewLog(ipfs, identities[1], &log.NewLogOptions{ ID: "X" })
				c.So(err, ShouldBeNil)


				for i := 1; i <= 5; i++ {
					_, err = logA.Append([]byte(fmt.Sprintf("entryA%d", i)), nextPointersAmount)
					c.So(err, ShouldBeNil)

					_, err = logB.Append([]byte(fmt.Sprintf("entryB%d", i)), nextPointersAmount)
					c.So(err, ShouldBeNil)
				}

				_, err = log3.Join(logA, -1)
				c.So(err, ShouldBeNil)

				_, err = log3.Join(logB, -1)
				c.So(err, ShouldBeNil)

				for i := 6; i <= 10; i++ {
					_, err = logA.Append([]byte(fmt.Sprintf("entryA%d", i)), nextPointersAmount)
					c.So(err, ShouldBeNil)
				}

				_, err = l.Join(log3, -1)
				c.So(err, ShouldBeNil)

				_, err = l.Append([]byte("entryC0"), nextPointersAmount)
				c.So(err, ShouldBeNil)

				_, err = l.Join(logA, -1)
				c.So(err, ShouldBeNil)

				hash, err := l.ToMultihash()

				// First 5
				res, err := log.NewFromMultihash(ipfs, identities[1], hash, &log.NewLogOptions{}, &log.FetchOptions{ Length: intPtr(5) })
				c.So(err, ShouldBeNil)

				first5 := []string{
					"entryA5", "entryB5", "entryC0", "entryA9", "entryA10",
				}

				c.So(entriesAsStrings(res.Values()), ShouldResemble, first5)

				// First 11
				res, err = log.NewFromMultihash(ipfs, identities[1], hash, &log.NewLogOptions{}, &log.FetchOptions{ Length: intPtr(11) })
				c.So(err, ShouldBeNil)

				first11 := []string{
					"entryA3", "entryB3", "entryA4", "entryB4",
					"entryA5", "entryB5",
					"entryC0",
					"entryA7", "entryA8", "entryA9", "entryA10",
        		}

				c.So(entriesAsStrings(res.Values()), ShouldResemble, first11)

				// All but one
				res, err = log.NewFromMultihash(ipfs, identities[1], hash, &log.NewLogOptions{}, &log.FetchOptions{ Length: intPtr(16-1) })
				c.So(err, ShouldBeNil)

				all := []string{
					"entryA1", /* excl */ "entryA2", "entryB2", "entryA3", "entryB3",
					"entryA4", "entryB4", "entryA5", "entryB5",
					"entryA6",
					"entryC0", "entryA7", "entryA8", "entryA9", "entryA10",
				}

				c.So(entriesAsStrings(res.Values()), ShouldResemble, all)

			})

			c.Convey("retrieves partially joined log deterministically - multiple next pointers", FailureHalts, func(c C) {
				nextPointersAmount := 64

				logA, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ ID:"X" })
				c.So(err, ShouldBeNil)
				logB, err := log.NewLog(ipfs, identities[2], &log.NewLogOptions{ ID:"X" })
				c.So(err, ShouldBeNil)
				log3, err := log.NewLog(ipfs, identities[3], &log.NewLogOptions{ ID: "X" })
				c.So(err, ShouldBeNil)
				l, err := log.NewLog(ipfs, identities[1], &log.NewLogOptions{ ID: "X" })
				c.So(err, ShouldBeNil)

				for i := 1; i <= 5; i++ {
					_, err = logA.Append([]byte(fmt.Sprintf("entryA%d", i)), nextPointersAmount)
					c.So(err, ShouldBeNil)

					_, err = logB.Append([]byte(fmt.Sprintf("entryB%d", i)), nextPointersAmount)
					c.So(err, ShouldBeNil)
				}

				_, err = log3.Join(logA, -1)
				c.So(err, ShouldBeNil)

				_, err = log3.Join(logB, -1)
				c.So(err, ShouldBeNil)

				for i := 6; i <= 10; i++ {
					_, err = logA.Append([]byte(fmt.Sprintf("entryA%d", i)), nextPointersAmount)
					c.So(err, ShouldBeNil)
				}

				_, err = l.Join(log3, -1)
				c.So(err, ShouldBeNil)

				_, err = l.Append([]byte("entryC0"), nextPointersAmount)
				c.So(err, ShouldBeNil)

				_, err = l.Join(logA, -1)
				c.So(err, ShouldBeNil)

				hash, err := l.ToMultihash()

				// First 5
				res, err := log.NewFromMultihash(ipfs, identities[1], hash, &log.NewLogOptions{}, &log.FetchOptions{ Length: intPtr(5) })
				c.So(err, ShouldBeNil)

				first5 := []string{
					"entryC0", "entryA7", "entryA8", "entryA9", "entryA10",
				}

				c.So(entriesAsStrings(res.Values()), ShouldResemble, first5)

				// First 11
				res, err = log.NewFromMultihash(ipfs, identities[1], hash, &log.NewLogOptions{}, &log.FetchOptions{ Length: intPtr(11) })
				c.So(err, ShouldBeNil)

				first11 := []string{
					"entryA1", "entryA2", "entryA3", "entryA4",
					"entryA5", "entryA6",
					"entryC0",
					"entryA7", "entryA8", "entryA9", "entryA10",
				}

				c.So(entriesAsStrings(res.Values()), ShouldResemble, first11)

				// All but one
				res, err = log.NewFromMultihash(ipfs, identities[1], hash, &log.NewLogOptions{}, &log.FetchOptions{ Length: intPtr(16-1) })
				c.So(err, ShouldBeNil)

				all := []string{
					"entryA1", /* excl */ "entryA2", "entryB2", "entryA3", "entryB3",
					"entryA4", "entryB4", "entryA5", "entryB5",
					"entryA6",
					"entryC0", "entryA7", "entryA8", "entryA9", "entryA10",
				}

				c.So(entriesAsStrings(res.Values()), ShouldResemble, all)
			})

			c.Convey("throws an error if ipfs is not defined", FailureHalts, func(c C) {
				_, err := log.NewFromEntry(nil, identities[0], []*entry.Entry{}, &log.NewLogOptions{ID: "X"}, &entry.FetchOptions{})
				c.So(err, ShouldNotBeNil)
				c.So(err.Error(), ShouldContainSubstring, "ipfs instance not defined")
			})

			c.Convey("fetches a log", FailureHalts, func (c C) {
				const amount = 100

				ts := time.Now().UnixNano() / 1000
				log1, err := log.NewLog(ipfs, identities[0], &log.NewLogOptions{ ID: "X" })
				c.So(err, ShouldBeNil)

				log2, err := log.NewLog(ipfs, identities[1], &log.NewLogOptions{ ID: "X" })
				c.So(err, ShouldBeNil)

				log3, err := log.NewLog(ipfs, identities[2], &log.NewLogOptions{ ID: "X" })
				c.So(err, ShouldBeNil)

				var items1 []*entry.Entry
				var items2 []*entry.Entry
				var items3 []*entry.Entry

				for i := 1; i <= amount; i++ {
					var nexts []cid.Cid
					prev1 := lastEntry(items1)
					prev2 := lastEntry(items2)
					prev3 := lastEntry(items3)

					if prev1 != nil {
						nexts = []cid.Cid{prev1.Hash}
					}

					n1, err := entry.CreateEntry(ipfs, log1.Identity, &entry.Entry{LogID: log1.ID, Payload: []byte(fmt.Sprintf("entryA%d-%d", i, ts)), Next: nexts}, log1.Clock)
					c.So(err, ShouldBeNil)

					nexts = []cid.Cid{n1.Hash}
					if prev2 != nil {
						nexts = []cid.Cid{prev2.Hash, n1.Hash}
					}

					n2, err := entry.CreateEntry(ipfs, log2.Identity, &entry.Entry{LogID: log2.ID, Payload: []byte(fmt.Sprintf("entryB%d-%d", i, ts)), Next: nexts}, log2.Clock)
					c.So(err, ShouldBeNil)

					nexts = []cid.Cid{n1.Hash, n2.Hash}
					if prev2 != nil {
						nexts = []cid.Cid{prev3.Hash, n1.Hash, n2.Hash}
					}

					n3, err := entry.CreateEntry(ipfs, log3.Identity, &entry.Entry{LogID: log3.ID, Payload: []byte(fmt.Sprintf("entryC%d-%d", i, ts)), Next: nexts}, log3.Clock)
					c.So(err, ShouldBeNil)

					log1.Clock.Tick()
					log2.Clock.Tick()
					log3.Clock.Tick()
					log1.Clock.Merge(log2.Clock)
					log1.Clock.Merge(log3.Clock)
					log2.Clock.Merge(log1.Clock)
					log2.Clock.Merge(log3.Clock)
					log3.Clock.Merge(log1.Clock)
					log3.Clock.Merge(log2.Clock)
					items1 = append(items1, n1)
					items2 = append(items2, n2)
					items3 = append(items3, n3)
				}

				c.Convey("returns all entries - no excluded entries", FailureHalts, func (c C) {
					a, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{ Length: intPtr(-1) })
					c.So(err, ShouldBeNil)

					c.So(a.Values().Len(), ShouldEqual, amount)
					c.So(a.Values().At(0).Hash.String(), ShouldEqual, items1[0].Hash.String())
				})

				c.Convey("returns all entries - including excluded entries", FailureHalts, func (c C) {
					// One entry
					a, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{ Exclude: []*entry.Entry{items1[0]}, Length: intPtr(-1) })
					c.So(err, ShouldBeNil)

					c.So(a.Values().Len(), ShouldEqual, amount)
					c.So(a.Values().At(0).Hash.String(), ShouldEqual, items1[0].Hash.String())

					// All entries
					b, err := log.NewFromEntry(ipfs, identities[0], []*entry.Entry{lastEntry(items1)}, &log.NewLogOptions{}, &entry.FetchOptions{ Exclude: items1, Length: intPtr(-1) })
					c.So(err, ShouldBeNil)

					c.So(b.Values().Len(), ShouldEqual, amount)
					c.So(b.Values().At(0).Hash.String(), ShouldEqual, items1[0].Hash.String())
				})
			})
		})
	})
}
