// Package evidence is the LeX side of the Context evidence plane. It freezes
// caller input into one Frozen state and verifies a caller-frozen state by
// asking Context to rebuild it. Retrieval, selection, budgeting, and rejection
// belong to Context; this package never recomputes them.
package evidence

import (
	"context"
	"fmt"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
)

// InitialMaxBytes limits Context source input for the initial LeX profile.
const InitialMaxBytes = 256 << 10

// BuildPack constructs a bounded frozen Context pack from authenticated,
// versioned source text. It performs no network, filesystem, or model I/O.
func BuildPack(
	ctx context.Context,
	projectID string,
	sources []contextmemory.Source,
	request contextmemory.PackRequest,
) (contextmemory.PackResult, error) {
	if request.ProjectID != projectID {
		return contextmemory.PackResult{}, fmt.Errorf("pack request project does not match authenticated project")
	}
	runtime, err := newRuntime(ctx, projectID, sources)
	if err != nil {
		return contextmemory.PackResult{}, err
	}
	result, err := runtime.ContextPack(ctx, request)
	if err != nil {
		return contextmemory.PackResult{}, fmt.Errorf("build Context pack: %w", err)
	}
	return result, nil
}

func newRuntime(ctx context.Context, projectID string, sources []contextmemory.Source) (*contextmemory.Runtime, error) {
	runtime, err := contextmemory.New(ctx, contextmemory.Config{
		ProjectID: projectID,
		Sources:   sources,
		MaxBytes:  InitialMaxBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("create Context runtime: %w", err)
	}
	return runtime, nil
}
