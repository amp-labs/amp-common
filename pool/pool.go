package pool

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/amp-labs/amp-common/should"
	"github.com/amp-labs/amp-common/try"
	"go.uber.org/atomic"
)

const (
	getTimeout       = 5 * time.Second
	putTimeout       = 10 * time.Second
	closeIdleTimeout = 30 * time.Second

	tickerFrequency = 30 * time.Second
)

var ErrTimeout = errors.New("timeout")

// Pool represents a pool of Objects. Objects must implement
// the io.Closer interface so that they can be safely discarded
// once they're no longer needed.
type Pool[C io.Closer] interface {
	// Get will fetch an object from the pool. If there's
	// an existing one, it will use that. If there's none
	// available, it will create a new one and return that.
	Get() (C, error)

	// Put will return an object to the pool.
	Put(c C)

	// CloseIdle will close and remove idle objects
	// from the pool. Idle is defined as not in-use
	// for a certain period of time (configurable).
	// It returns the number of objects closed, or
	// an error if any of the close calls failed.
	CloseIdle(minTimeIdle time.Duration) (int, error)

	// Close will close the entire pool and close all objects.
	Close() error
}

// getRequest is used internally to request an object from the pool.
type getRequest[C io.Closer] struct {
	resultChan chan try.Try[C]
}

// putRequest is used internally to return an object to the pool.
type putRequest[C io.Closer] struct {
	doneChan chan struct{}
	obj      C
}

// closeIdleResponse is used internally to respond to closeIdleRequest.
type closeIdleResponse struct {
	errs      []error
	successes int
}

// closeIdleRequest is used internally to request closing idle objects.
type closeIdleRequest struct {
	minIdle  time.Duration
	doneChan chan closeIdleResponse
}

type poolOptions struct {
	name string
}

type Option func(*poolOptions)

func WithName(name string) Option {
	return func(p *poolOptions) {
		p.name = name
	}
}

// New will create a new Pool which will grow dynamically as demand
// increases. All objects are kept indefinitely, until CloseIdle or
// Close is called.
func New[C io.Closer](factory func() (C, error), opts ...Option) Pool[C] {
	options := &poolOptions{
		name: "pool",
	}

	for _, opt := range opts {
		opt(options)
	}

	poolInst := &poolImpl[C]{
		name:        options.name,
		getCh:       make(chan getRequest[C]),
		putCh:       make(chan putRequest[C]),
		ciCh:        make(chan closeIdleRequest),
		closeCh:     make(chan error, 1),
		create:      factory,
		running:     atomic.NewBool(true),
		outstanding: atomic.NewInt64(0),
	}

	go poolInst.loop()

	poolAlive.WithLabelValues(poolInst.name).Set(1)
	poolObjectsTotal.WithLabelValues(poolInst.name).Set(0)
	poolObjectsInUse.WithLabelValues(poolInst.name).Set(0)
	poolObjectsIdle.WithLabelValues(poolInst.name).Set(0)
	objectsClosed.WithLabelValues(poolInst.name).Add(0)
	objectsClosedErrors.WithLabelValues(poolInst.name).Add(0)
	objectsCreated.WithLabelValues(poolInst.name).Add(0)
	creationErrors.WithLabelValues(poolInst.name).Add(0)

	poolCreated.WithLabelValues(poolInst.name).Inc()

	return poolInst
}

type poolImpl[C io.Closer] struct {
	name    string
	create  func() (C, error)
	getCh   chan getRequest[C]
	putCh   chan putRequest[C]
	ciCh    chan closeIdleRequest
	drain   sync.WaitGroup
	closeCh chan error

	outstanding *atomic.Int64
	running     *atomic.Bool
}

func (g *poolImpl[C]) createObject() (C, error) {
	obj, err := g.create()
	if err != nil {
		creationErrors.WithLabelValues(g.name).Inc()

		return obj, err
	} else {
		objectsCreated.WithLabelValues(g.name).Inc()
		poolObjectsTotal.WithLabelValues(g.name).Inc()

		return obj, nil
	}
}

type poolObject[C io.Closer] struct {
	obj         C
	lastTouched time.Time
}

func (g *poolImpl[C]) handleGet(get getRequest[C], objectPool *[]poolObject[C]) {
	if len(*objectPool) > 0 {
		obj := (*objectPool)[0]
		*objectPool = (*objectPool)[1:]
		get.resultChan <- try.Try[C]{Value: obj.obj}

		g.outstanding.Inc()
		poolObjectsInUse.WithLabelValues(g.name).Inc()
		poolObjectsIdle.WithLabelValues(g.name).Dec()
		close(get.resultChan)
	} else {
		obj, err := g.createObject()
		get.resultChan <- try.Try[C]{
			Value: obj,
			Error: err,
		}

		if err == nil {
			g.outstanding.Inc()
			poolObjectsInUse.WithLabelValues(g.name).Inc()
		}

		close(get.resultChan)
	}
}

func (g *poolImpl[C]) handlePut(put putRequest[C], objectPool *[]poolObject[C]) {
	*objectPool = append(*objectPool, poolObject[C]{
		obj:         put.obj,
		lastTouched: time.Now(),
	})

	g.outstanding.Dec()
	poolObjectsInUse.WithLabelValues(g.name).Dec()
	poolObjectsIdle.WithLabelValues(g.name).Inc()
	put.doneChan <- struct{}{}
	close(put.doneChan)
}

func (g *poolImpl[C]) handleCloseIdle(closeIdleReq closeIdleRequest, objectPool *[]poolObject[C]) closeIdleResponse {
	var errs []error

	purged := 0

	var remainder []poolObject[C]

	for _, obj := range *objectPool {
		age := time.Since(obj.lastTouched)
		if age < closeIdleReq.minIdle {
			remainder = append(remainder, obj)

			continue
		}

		if err := obj.obj.Close(); err != nil { //nolint:typecheck
			errs = append(errs, err)
			remainder = append(remainder, obj)

			objectsClosedErrors.WithLabelValues(g.name).Inc()
		} else {
			poolObjectsTotal.WithLabelValues(g.name).Dec()
			poolObjectsIdle.WithLabelValues(g.name).Dec()
			objectsClosed.WithLabelValues(g.name).Inc()

			purged++
		}
	}

	*objectPool = remainder

	return closeIdleResponse{
		errs:      errs,
		successes: purged,
	}
}

func (g *poolImpl[C]) loop() {
	defer g.running.Store(false)
	defer poolAlive.WithLabelValues(g.name).Set(0)

	var objectPool []poolObject[C]

	done := 0

	var drainOnce sync.Once

	stopDrain := func() {
		drainOnce.Do(func() {
			g.drain.Done()
		})
	}

	defer stopDrain()

	ticker := time.NewTimer(tickerFrequency)
	defer ticker.Stop()

	for {
		select {
		case get, ok := <-g.getCh:
			if ok {
				g.handleGet(get, &objectPool)
			} else {
				done++
			}
		case put, ok := <-g.putCh:
			if ok {
				g.handlePut(put, &objectPool)
			} else {
				done++
			}
		case closeIdleReq, ok := <-g.ciCh:
			if ok {
				resp := g.handleCloseIdle(closeIdleReq, &objectPool)
				closeIdleReq.doneChan <- resp
			} else {
				done++
			}
		case <-ticker.C:
		}

		outstanding := g.outstanding.Load()

		if done >= 1 && outstanding > 0 {
			continue
		}

		if (done == 1 || done == 2) && outstanding == 0 {
			stopDrain()

			continue
		}

		const allChannelsClosed = 3
		if done >= allChannelsClosed {
			break
		}
	}

	var errs []error

	for _, d := range objectPool {
		if err := d.obj.Close(); err != nil { //nolint:typecheck
			errs = append(errs, err)

			objectsClosedErrors.WithLabelValues(g.name).Inc()
		} else {
			objectsClosed.WithLabelValues(g.name).Inc()
		}
	}

	poolObjectsTotal.WithLabelValues(g.name).Set(0)
	poolObjectsIdle.WithLabelValues(g.name).Set(0)
	poolObjectsInUse.WithLabelValues(g.name).Set(0)

	g.closeCh <- joinErrors(errs...)
	close(g.closeCh)
}

func joinErrors(errs ...error) error {
	switch {
	case len(errs) == 0:
		return nil
	case len(errs) == 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}

var ErrPoolClosed = errors.New("pool is closed")

var ErrPoolGet = errors.New("unable to run poolImpl.Get, perhaps the channel is closed")

// Get will fetch an object from the pool. If there's
// an existing one, it will use that. If there's none
// available, it will create a new one and return that.
func (g *poolImpl[C]) Get() (obj C, err error) {
	if !g.running.Load() {
		var zero C

		return zero, ErrPoolClosed
	}

	rsp := make(chan try.Try[C], 1)

	defer func() {
		if tmp := recover(); tmp != nil {
			err = fmt.Errorf("%w: %v", ErrPoolGet, tmp)

			var zero C
			obj = zero
		}
	}()

	timeoutTimer := time.NewTimer(getTimeout)
	defer timeoutTimer.Stop()

	r := getRequest[C]{resultChan: rsp}

	select {
	case g.getCh <- r:
		select {
		case rs := <-rsp:
			return rs.Value, rs.Error
		case <-timeoutTimer.C:
			slog.Warn("pool.Get has timed out, creating a new object")

			inst, err := g.createObject()
			if err == nil {
				g.outstanding.Inc()
			}

			return inst, err
		}
	case <-timeoutTimer.C:
		slog.Warn("pool.Get has timed out, creating a new object")

		inst, err := g.createObject()
		if err == nil {
			g.outstanding.Inc()
		}

		return inst, err
	}
}

// Put will return an object to the pool.
func (g *poolImpl[C]) Put(obj C) {
	if !g.running.Load() {
		slog.Error("pool is closed, unable to put object back")

		return
	}

	rsp := make(chan struct{}, 1)

	defer func() {
		if tmp := recover(); tmp != nil {
			slog.Error("unable to run poolImpl.Put, perhaps the channel is closed", "error", tmp)
		}
	}()

	timeoutTimer := time.NewTimer(putTimeout)
	defer timeoutTimer.Stop()

	req := putRequest[C]{
		obj:      obj,
		doneChan: rsp,
	}

	select {
	case g.putCh <- req:
		select {
		case <-rsp:
			return
		case <-timeoutTimer.C:
			return
		}
	case <-timeoutTimer.C:
		should.Close(obj, "unable to close pool object")
	}
}

// CloseIdle will close and remove idle objects
// from the pool. Idle is defined as not in-use
// for a certain period of time (configurable).
// It returns the number of objects closed, or
// an error if any of the close calls failed.
func (g *poolImpl[C]) CloseIdle(minTimeIdle time.Duration) (int, error) {
	if !g.running.Load() {
		return 0, ErrPoolClosed
	}

	rsp := make(chan closeIdleResponse, 1)

	defer func() {
		if tmp := recover(); tmp != nil {
			slog.Error("unable to run poolImpl.CloseIdle, perhaps the channel is closed", "error", tmp)
		}
	}()

	timeoutTimer := time.NewTimer(closeIdleTimeout)
	defer timeoutTimer.Stop()

	req := closeIdleRequest{
		minIdle:  minTimeIdle,
		doneChan: rsp,
	}

	select {
	case g.ciCh <- req:
		select {
		case r := <-rsp:
			return r.successes, joinErrors(r.errs...)
		case <-timeoutTimer.C:
			return 0, ErrTimeout
		}
	case <-timeoutTimer.C:
		return 0, ErrTimeout
	}
}

// Close will close the entire pool and close all pooled objects.
func (g *poolImpl[C]) Close() error {
	if !g.running.Load() {
		return ErrPoolClosed
	}

	g.drain.Add(1)
	close(g.getCh)

	g.drain.Wait()
	close(g.putCh)

	close(g.ciCh)

	return <-g.closeCh
}
