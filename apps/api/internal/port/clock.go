// Package port declares the secondary ports the use cases depend on: what they need from the outside
// world (time, storage, rates, email), implemented by adapters and injected in cmd.
package port

import "time"

// Clock tells the current instant. It is the only source of time for the use cases; internal/domain and
// internal/app never call time.Now, and lint enforces it.
type Clock interface {
	Now() time.Time
}
