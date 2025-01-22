package github

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

// debugTransport wraps an HTTP transport and logs requests/responses
type debugTransport struct {
	transport http.RoundTripper
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Dump request
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, fmt.Errorf("failed to dump request: %w", err)
	}
	fmt.Printf(">>> Request:\n%s\n", string(reqDump))

	// Execute request
	resp, err := d.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Dump response
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("failed to dump response: %w", err)
	}
	fmt.Printf("<<< Response:\n%s\n", string(respDump))

	return resp, nil
}
