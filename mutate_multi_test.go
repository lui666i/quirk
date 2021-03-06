package quirk

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/dgraph-io/dgo/v2"
	. "github.com/franela/goblin"
)

func TestMutateMulti(t *testing.T) {
	g := Goblin(t)
	c := NewClient()

	g.Describe("mutateMulti", func() {
		ctx := context.Background()

		g.It("Should not error", func() {
			g.Assert(c.mutateMulti(ctx,
				dgo.NewDgraphClient(&testDgraphClient{}), []interface{}{},
				make(map[string]UID), c.mutateSingleStruct)).
				Equal(nil)
		})
	})
}

func TestLaunchWorkers(t *testing.T) {
	g := Goblin(t)

	g.Describe("mutateMultiStruct", func() {
		var (
			done = make(chan error)
			quit = make(chan bool)
		)

		g.It("should not error", func() {
			g.Assert(launchWorkers(0, &pb.ProgressBar{}, done, quit)).
				Equal(nil)
		})
	})
}

func TestMutationWorker(t *testing.T) {
	g := Goblin(t)

	g.Describe("mutation worker", func() {
		var (
			m      sync.Mutex
			mSS    = NewClient(WithPredicateKey("username")).mutateSingleStruct
			ctx    = context.Background()
			api    = &testDgraphClient{queryResponse: []byte("{}")}
			dg     = dgo.NewDgraphClient(api)
			logger = NewNilLogger()
			uidMap = make(map[string]UID)
			done   = make(chan error, 1)
		)

		g.It("should not error when new", func() {
			read := make(chan interface{})
			quit := make(chan bool)

			api.shouldAbort = false
			// oof that's a lot of parameters...
			// Hello past self, don't worry I got your back covered.
			pkg := &workerPackage{dg, &m, mSS, logger, &pb.ProgressBar{}}
			go mutationWorker(ctx, pkg, uidMap, read, quit, done)

			// So then the logging if statement passes.
			time.Sleep(200 * time.Millisecond)

			close(quit)
			time.Sleep(100 * time.Millisecond)
			read <- &testPersonCorrect
			read <- &testPersonCorrect

			close(read)
		})

		g.It("should not error when old", func() {
			read := make(chan interface{})
			quit := make(chan bool)

			api.shouldAbort = true

			// oof that's a lot of parameters...
			// Hello past self, don't worry I got your back covered.
			pkg := &workerPackage{dg, &m, mSS, logger, &pb.ProgressBar{}}
			go mutationWorker(ctx, pkg, uidMap, read, quit, done)

			read <- &testPersonCorrect

			// So then the set to false won't come in too early.
			time.Sleep(100 * time.Millisecond)
			api.shouldAbort = false

			close(read)

			err := <-done

			g.Assert(err).Equal(nil)
		})
	})
}
