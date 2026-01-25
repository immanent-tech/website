// Copyright 2025 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package main

import "sync"

type stateTracker struct {
	sync.Mutex

	values map[string]any
}

var state *stateTracker

var loadState = sync.OnceFunc(func() {
	state = &stateTracker{
		values: make(map[string]any),
	}
})

func setState(key string, value any) {
	loadState()
	state.Lock()
	defer state.Unlock()

	state.values[key] = value
}

func getState[T any](key string) T {
	loadState()
	state.Lock()
	defer state.Unlock()

	return state.values[key].(T)
}
