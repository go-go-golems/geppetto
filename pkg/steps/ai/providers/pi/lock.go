package pi

import (
	"context"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// fileLock is compatible with proper-lockfile, which Pi uses for auth.json:
//
//   - the lock is a directory "<file>.lock", created with mkdir (atomic);
//   - its holder refreshes the directory's mtime while it holds it;
//   - another process may remove a lock whose mtime is older than its stale
//     interval, then retry.
//
// Pi's async lock uses a 30 s stale interval and its sync lock proper-lockfile's
// default of 10 s, so the mtime is refreshed every 5 s: neither sees a held lock
// as stale. A lock whose mtime changes under us, or that disappears, is
// compromised; the write is then abandoned.
type fileLock struct {
	path     string
	stale    time.Duration
	update   time.Duration
	deadline time.Duration
}

func newFileLock(file string) fileLock {
	return fileLock{path: file + ".lock", stale: 30 * time.Second, update: 5 * time.Second, deadline: 30 * time.Second}
}

type heldLock struct {
	l           fileLock
	mu          sync.Mutex
	mtime       time.Time
	compromised error
	stop        chan struct{}
	done        chan struct{}
}

// acquire takes the lock, waiting with jittered backoff up to the deadline.
func (l fileLock) acquire(ctx context.Context) (*heldLock, error) {
	end := time.Now().Add(l.deadline)
	for retry := 0; ; retry++ {
		err := os.Mkdir(l.path, 0o700)
		if err == nil {
			return l.hold()
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, errors.Wrap(err, "create auth lock")
		}
		if st, serr := os.Stat(l.path); serr == nil && time.Since(st.ModTime()) > l.stale {
			// Stale: remove and retry at once, as proper-lockfile does.
			if rerr := os.Remove(l.path); rerr == nil || errors.Is(rerr, os.ErrNotExist) {
				continue
			}
		}
		if time.Now().After(end) {
			return nil, errors.New("auth.json is locked by another process")
		}
		delay := min(10*time.Millisecond<<min(retry, 7), time.Second)
		delay = delay + time.Duration(rand.Int64N(int64(delay))) // #nosec G404 -- jitter
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (l fileLock) hold() (*heldLock, error) {
	now := time.Now()
	if err := os.Chtimes(l.path, now, now); err != nil {
		_ = os.Remove(l.path)
		return nil, errors.Wrap(err, "touch auth lock")
	}
	st, err := os.Stat(l.path)
	if err != nil {
		return nil, errors.Wrap(err, "stat auth lock")
	}
	h := &heldLock{l: l, mtime: st.ModTime(), stop: make(chan struct{}), done: make(chan struct{})}
	go h.refresh()
	return h, nil
}

func (h *heldLock) refresh() {
	defer close(h.done)
	t := time.NewTicker(h.l.update)
	defer t.Stop()
	for {
		select {
		case <-h.stop:
			return
		case <-t.C:
		}
		h.mu.Lock()
		st, err := os.Stat(h.l.path)
		switch {
		case err != nil:
			h.compromised = errors.New("auth lock was removed")
		case !st.ModTime().Equal(h.mtime):
			h.compromised = errors.New("auth lock was taken over")
		default:
			now := time.Now()
			if err := os.Chtimes(h.l.path, now, now); err == nil {
				if st, err := os.Stat(h.l.path); err == nil {
					h.mtime = st.ModTime()
				}
			}
		}
		bad := h.compromised != nil
		h.mu.Unlock()
		if bad {
			return
		}
	}
}

// check reports a compromised lock.
func (h *heldLock) check() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.compromised
}

// release stops refreshing and removes the lock (unless it was taken over).
func (h *heldLock) release() {
	close(h.stop)
	<-h.done
	if h.check() == nil {
		_ = os.Remove(h.l.path)
	}
}
