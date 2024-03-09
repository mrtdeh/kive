package watcher

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type Filter struct {
	open      bool
	watchFunc func(any) bool
	signalCh  chan struct{}
}
type Watcher struct {
	value      any
	valueMutex sync.Mutex
	distract   chan struct{}
	filters    map[string]*Filter
}

func NewWatcher(initialValue any) *Watcher {
	return &Watcher{
		value:    initialValue,
		filters:  make(map[string]*Filter),
		distract: make(chan struct{}),
	}
}

func (w *Watcher) AddFilter(id string, watchFunc func(any) bool) *Filter {
	w.valueMutex.Lock()
	defer w.valueMutex.Unlock()

	f := &Filter{
		open:      false,
		watchFunc: watchFunc,
		signalCh:  make(chan struct{}),
	}
	w.filters[id] = f

	cuurentChan := func() chan struct{} {
		if f.watchFunc != nil && f.watchFunc(w.value) {
			return f.signalCh
		}
		tmp := make(chan struct{}, 1)
		tmp <- struct{}{}
		return tmp
	}

	go func() {
		for {
			select {
			case <-w.distract:
			case cuurentChan() <- struct{}{}:
			}
		}
	}()

	return f
}

func (w *Watcher) Set(newValue any) {
	w.valueMutex.Lock()
	defer w.valueMutex.Unlock()

	w.value = newValue
	w.checkAndSendSignal()
}

func (w *Watcher) Value() any {
	w.valueMutex.Lock()
	defer w.valueMutex.Unlock()

	return w.value
}

func (w *Watcher) checkAndSendSignal() {
	// reset filters by new value
	for _, f := range w.filters {
		if f.watchFunc != nil && f.watchFunc(w.value) {
			activeChan(f.signalCh)
			f.open = true
		} else {
			f.open = false
			zeroChan(f.signalCh)
		}
	}

	// refresh repeated goroutine
	close(w.distract)
	w.distract = make(chan struct{})
}

// register filter for this values if necessary
func (w *Watcher) intilizeValues(vals []string) *Filter {
	var f *Filter
	// generate id from input values
	sort.Slice(vals, func(i, j int) bool {
		return vals[i] < vals[j]
	})
	id := strings.Join(vals, "")
	// add filter by id if not already present
	if f = w.filters[id]; f == nil {
		f = w.AddFilter(id, func(a any) bool {
			for _, i := range vals {
				if i == a {
					return true
				}
			}
			return false
		})
	}

	return f
}

// return a channel that fire on input is match with current state
func (w *Watcher) On(ids ...string) <-chan struct{} {
	f := w.intilizeValues(ids)

	return f.signalCh
}

// block thread on input is match with current state
func (w *Watcher) WaitFor(ctx context.Context, ids ...string) {
	f := w.intilizeValues(ids)

	select {
	case <-ctx.Done():
	case <-f.signalCh:
		activeChan(f.signalCh)
	}

}

func activeChan(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

func zeroChan(c chan struct{}) {
	select {
	case <-c:
	default:
	}
}
