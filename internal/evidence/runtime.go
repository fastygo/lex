// Package evidence adapts the pinned Context embedded runtime to LeX.
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

	runtime, err := contextmemory.New(ctx, contextmemory.Config{
		ProjectID: projectID,
		Sources:   sources,
		MaxBytes:  InitialMaxBytes,
	})
	if err != nil {
		return contextmemory.PackResult{}, fmt.Errorf("create Context runtime: %w", err)
	}
	result, err := runtime.ContextPack(ctx, request)
	if err != nil {
		return contextmemory.PackResult{}, fmt.Errorf("build Context pack: %w", err)
	}
	return result, nil
}
