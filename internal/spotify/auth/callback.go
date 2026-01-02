package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// CallbackResult contains the result of the OAuth callback.
type CallbackResult struct {
	Code  string
	State string
	Error string
}

// CallbackServer handles the OAuth callback from Spotify.
type CallbackServer struct {
	server   *http.Server
	listener net.Listener
	result   chan CallbackResult
}

// NewCallbackServer creates a new callback server on the specified port.
func NewCallbackServer(port int) (*CallbackServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	cs := &CallbackServer{
		listener: listener,
		result:   make(chan CallbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", cs.handleCallback)

	cs.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return cs, nil
}

// Start begins serving HTTP requests in the background.
func (cs *CallbackServer) Start() {
	go func() {
		_ = cs.server.Serve(cs.listener)
	}()
}

// Wait blocks until a callback is received or context is cancelled.
// Returns the callback result or an error if the context times out.
func (cs *CallbackServer) Wait(ctx context.Context) (CallbackResult, error) {
	select {
	case result := <-cs.result:
		return result, nil
	case <-ctx.Done():
		return CallbackResult{}, ctx.Err()
	}
}

// Shutdown gracefully shuts down the server.
func (cs *CallbackServer) Shutdown(ctx context.Context) error {
	return cs.server.Shutdown(ctx)
}

// Port returns the port the server is listening on.
func (cs *CallbackServer) Port() int {
	return cs.listener.Addr().(*net.TCPAddr).Port
}

func (cs *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	result := CallbackResult{
		Code:  query.Get("code"),
		State: query.Get("state"),
		Error: query.Get("error"),
	}

	// Send result (non-blocking in case of duplicate callbacks)
	select {
	case cs.result <- result:
	default:
	}

	// Respond to the browser
	if result.Error != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
<h1>Authentication Failed</h1>
<p>Error: %s</p>
<p>You can close this window.</p>
</body>
</html>`, result.Error)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body>
<h1>Authentication Successful</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)
}
