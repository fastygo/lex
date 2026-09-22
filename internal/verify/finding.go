// Package verify provides deterministic LeX verification primitives.
package verify

// Finding is one failed deterministic obligation. It carries no domain verdict:
// meaning and downstream action belong to the caller.
type Finding struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}
