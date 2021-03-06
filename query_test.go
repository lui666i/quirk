package quirk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/dgraph-io/dgo/v2"
	. "github.com/franela/goblin"
)

func BenchmarkCreateQuery(b *testing.B) {
	var builder strings.Builder

	for i := 0; i < b.N; i++ {
		_ = createQuery(&builder, testPredValCorrect)
	}
}

func BenchmarkExecuteQuery(b *testing.B) {
	var (
		ctx     = context.Background()
		decoder = make(queryDecode)
		dgraph  = dgo.NewDgraphClient(&testDgraphClient{
			queryResponse: testValidJSONOutput,
		})
	)

	for i := 0; i < b.N; i++ {
		_ = executeQuery(ctx, dgraph.NewTxn(),
			&testBuilder{}, &DupleNode{}, &decoder)
	}
}

func BenchmarkFindDecodedUID(b *testing.B) {
	uid := "0x1"
	emptyDecode := make(queryDecode)

	emptyDecode["a"] = []struct{ UID *string }{
		{UID: &uid},
	}

	for i := 0; i < b.N; i++ {
		_, _ = findDecodedUID(emptyDecode)
	}
}

func TestQueryUID(t *testing.T) {
	g := Goblin(t)

	g.Describe("createUID", func() {
		ctx := context.Background()
		dgraph := dgo.NewDgraphClient(&testDgraphClient{
			queryResponse: testValidJSONOutput,
		})

		g.It("Should not error with nil parameters", func() {
			s, err := queryUID(ctx, dgraph.NewTxn(),
				&testBuilder{}, &DupleNode{})

			g.Assert(s).Equal("0x1")
			g.Assert(err).Equal(nil)
		})

		g.It("Should error when executeQuery errors", func() {
			s, err := queryUID(ctx, dgraph.NewTxn(),
				&testBuilder{failOn: 1}, &DupleNode{})

			g.Assert(s).Equal("")
			g.Assert(err).Equal(errors.New("WRITE_ERROR"))
		})
	})
}

func TestFindDecodeUID(t *testing.T) {
	g := Goblin(t)

	g.Describe("findDecodeUID", func() {
		emptyDecode := make(queryDecode)

		g.It("Should not error with empty queryDecode", func() {
			s, err := findDecodedUID(emptyDecode)

			g.Assert(s).Equal("")
			g.Assert(err).Equal(error(nil))
		})

		g.It("Should error with empty struct slice", func() {
			emptyDecode["a"] = []struct{ UID *string }{
				{UID: new(string)},
				{UID: new(string)},
			}

			s, err := findDecodedUID(emptyDecode)

			g.Assert(s).Equal("")
			g.Assert(err).Equal(&QueryError{
				Msg: msgTooManyResponses, Function: "findDecodedUID"})
		})

		g.It("Should error with nil UID", func() {
			emptyDecode["a"] = []struct{ UID *string }{
				{UID: nil},
			}

			s, err := findDecodedUID(emptyDecode)

			g.Assert(s).Equal("")
			g.Assert(err).Equal(&QueryError{
				Msg: msgNilUID, Function: "findDecodedUID"})
		})

		g.It("Should return valid UID", func() {
			uid := "0x1"
			emptyDecode["a"] = []struct{ UID *string }{
				{UID: &uid},
			}

			s, err := findDecodedUID(emptyDecode)

			g.Assert(s).Equal(uid)
			g.Assert(err).Equal(error(nil))
		})
	})
}

func TestExecuteQuery(t *testing.T) {
	g := Goblin(t)

	g.Describe("executeQuery", func() {
		ctx := context.Background()
		emptyDecode := make(queryDecode)

		dgraph := dgo.NewDgraphClient(&testDgraphClient{
			queryResponse: testValidJSONOutput,
		})

		g.It("Should not error with valid parameters", func() {
			g.Assert(executeQuery(ctx, dgraph.NewTxn(),
				&testBuilder{}, &DupleNode{}, &emptyDecode)).
				Equal(error(nil))
		})

		g.It("Should error when builder fails first use", func() {
			g.Assert(executeQuery(ctx, dgraph.NewTxn(),
				&testBuilder{failOn: 1}, &DupleNode{}, &emptyDecode)).
				Equal(errors.New("WRITE_ERROR"))
		})

		g.It("Should not error when builder returns empty query", func() {
			g.Assert(executeQuery(ctx, dgraph.NewTxn(),
				&testBuilder{stringOutput: emptyQuery}, &DupleNode{}, &emptyDecode)).
				Equal(error(nil))
		})

		g.It("Should error when txn fails", func() {
			faildgraph := dgo.NewDgraphClient(&testDgraphClient{
				failQueryOn: 1,
				shouldAbort: true,
			})
			g.Assert(executeQuery(ctx, faildgraph.NewTxn(),
				&testBuilder{}, &DupleNode{}, &emptyDecode)).
				Equal(&QueryError{
					Function: "executeQuery",
					Query:    "",
					ExtErr:   errors.New("QUERY_ERROR"),
				})
		})

		g.It("Should error when txn fails", func() {
			faildgraph := dgo.NewDgraphClient(&testDgraphClient{
				shouldAbort: true,
			})
			err := executeQuery(ctx, faildgraph.NewTxn(),
				&testBuilder{}, &DupleNode{}, &emptyDecode)

			// Can't test the value of the json error.
			// Instead just test if the function returned an error at all.
			if err == nil {
				g.Fail(err)
			}
		})
	})
}

func TestCreateQuery(t *testing.T) {
	g := Goblin(t)

	g.Describe("createQuery", func() {

		g.It("Should not return an error", func(done Done) {
			go func() {
				g.Assert(createQuery(&strings.Builder{}, testPredValCorrect)).
					Equal(error(nil))
				done()
			}()
		})

		g.It("Should error when builder fails on first use", func(done Done) {
			go func() {
				g.Assert(createQuery(&testBuilder{failOn: 1}, &DupleNode{})).
					Equal(errors.New("WRITE_ERROR"))
				done()
			}()
		})

		g.It("Should error when builder fails when adding unique predicates", func(done Done) {
			go func() {
				g.Assert(createQuery(&testBuilder{failOn: 2}, testPredValCorrect)).
					Equal(&QueryError{
						ExtErr:   errors.New("WRITE_ERROR"),
						Msg:      fmt.Sprintf(msgBuilderWriting, "username", "damienstamates"),
						Function: "createQuery",
					})
				done()
			}()
		})

		g.It("Should error when builder fails on last use", func(done Done) {
			go func() {
				// When putting in empty predValPairs it will skip the for loop.
				// This is why the last use is set to 2.
				g.Assert(createQuery(&testBuilder{failOn: 2}, &DupleNode{})).
					Equal(errors.New("WRITE_ERROR"))
				done()
			}()
		})
	})
}
