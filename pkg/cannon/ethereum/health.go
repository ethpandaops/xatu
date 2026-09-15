package ethereum

import "time"

// syncWatchdog turns a series of Synced() results into two decisions: when the
// cannon should complain that its beacon node has been unusable for a while,
// and when it should stop waiting and exit so a supervisor can restart it.
type syncWatchdog struct {
	logEvery     time.Duration
	restartAfter time.Duration

	unhealthySince time.Time
	lastLogged     time.Time
}

// syncVerdict is the outcome of one watchdog observation.
type syncVerdict struct {
	unhealthyFor time.Duration
	shouldLog    bool
	shouldExit   bool
}

func newSyncWatchdog(logEvery, restartAfter time.Duration) *syncWatchdog {
	return &syncWatchdog{
		logEvery:     logEvery,
		restartAfter: restartAfter,
	}
}

// observe records the result of a Synced() check taken at now. The first log
// verdict comes once the node has been unhealthy for logEvery, then repeats at
// that cadence. A restartAfter of zero disables the exit verdict.
func (w *syncWatchdog) observe(now time.Time, healthy bool) syncVerdict {
	if healthy {
		w.unhealthySince = time.Time{}
		w.lastLogged = time.Time{}

		return syncVerdict{}
	}

	if w.unhealthySince.IsZero() {
		w.unhealthySince = now
	}

	verdict := syncVerdict{unhealthyFor: now.Sub(w.unhealthySince)}

	dueForLog := w.lastLogged.IsZero() || now.Sub(w.lastLogged) >= w.logEvery
	if verdict.unhealthyFor >= w.logEvery && dueForLog {
		verdict.shouldLog = true
		w.lastLogged = now
	}

	if w.restartAfter > 0 && verdict.unhealthyFor >= w.restartAfter {
		verdict.shouldExit = true
	}

	return verdict
}
