package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/filetoolsserver"
	"github.com/zoster81/mcp-file-tools/filetoolsserver/handler"
	"github.com/zoster81/mcp-file-tools/internal/config"
)

func TestHandlerHealthAndReadinessExposeNoSensitiveState(t *testing.T) {
	h := newTestHandler(t, 2)

	for _, test := range []struct {
		path   string
		ready  bool
		status int
		body   string
	}{
		{path: "/healthz", status: http.StatusOK, body: "ok\n"},
		{path: "/readyz", status: http.StatusServiceUnavailable, body: "not ready\n"},
		{path: "/readyz", ready: true, status: http.StatusOK, body: "ready\n"},
	} {
		h.setReady(test.ready)
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765"+test.path, nil)
		req.Host = "127.0.0.1:8765"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != test.status || rec.Body.String() != test.body {
			t.Fatalf("%s = %d %q, want %d %q", test.path, rec.Code, rec.Body.String(), test.status, test.body)
		}
		if strings.Contains(rec.Body.String(), "mcp-file-tools") || strings.Contains(rec.Body.String(), "root") {
			t.Fatalf("health response leaked details: %q", rec.Body.String())
		}
	}
}

func TestHandlerRejectsHostOriginAndMissingToken(t *testing.T) {
	h := newTestHandler(t, 2)
	h.setReady(true)

	tests := []struct {
		name   string
		host   string
		origin string
		token  string
		status int
	}{
		{name: "host", host: "attacker.example", token: testToken, status: http.StatusForbidden},
		{name: "origin", host: "127.0.0.1:8765", origin: "https://attacker.example", token: testToken, status: http.StatusForbidden},
		{name: "missing token", host: "127.0.0.1:8765", status: http.StatusUnauthorized},
		{name: "query token rejected", host: "127.0.0.1:8765", status: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url := "http://127.0.0.1:8765/mcp"
			if test.name == "query token rejected" {
				url += "?access_token=" + testToken
			}
			req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(`{}`))
			req.Host = test.host
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != test.status {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.status, rec.Body.String())
			}
			if test.status == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("401 response missing bearer challenge")
			}
		})
	}
}

func TestHandlerReauthenticatesEveryMCPMethod(t *testing.T) {
	h := newTestHandler(t, 2)
	h.setReady(true)

	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "http://127.0.0.1:8765/mcp", strings.NewReader(`{}`))
			req.Host = "127.0.0.1:8765"
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("missing bearer challenge")
			}
		})
	}
}

func TestHandlerRejectsWrongSchemeCookieAndUnknownSession(t *testing.T) {
	h := newTestHandler(t, 2)
	h.setReady(true)

	for _, test := range []struct {
		name       string
		authorize  string
		cookie     string
		sessionID  string
		wantStatus int
	}{
		{name: "wrong token", authorize: "Bearer " + testToken + "x", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", authorize: "Basic " + testToken, wantStatus: http.StatusUnauthorized},
		{name: "cookie ignored", cookie: testToken, wantStatus: http.StatusUnauthorized},
		{name: "unknown session", authorize: "Bearer " + testToken, sessionID: "unknown-session", wantStatus: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/mcp", nil)
			req.Host = "127.0.0.1:8765"
			if test.authorize != "" {
				req.Header.Set("Authorization", test.authorize)
			}
			if test.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "access_token", Value: test.cookie})
			}
			if test.sessionID != "" {
				req.Header.Set(sessionIDHeader, test.sessionID)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandlerAllowsConfiguredOriginWithoutEnablingCORS(t *testing.T) {
	cfg := validTestConfig(2)
	cfg.AllowedOrigins = map[string]struct{}{"https://app.example.test": {}}
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "origin-test"})
	h := NewHandler(cfg, server, nil)
	h.setReady(true)

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8765/mcp", strings.NewReader(`{}`))
	req.Host = "127.0.0.1:8765"
	req.Header.Set("Origin", "https://app.example.test")
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatalf("configured origin was rejected: %q", rec.Body.String())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("handler emitted CORS header %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestHandlerEnforcesTrustedProxyBoundary(t *testing.T) {
	cfg, err := LoadConfig(environment(map[string]string{
		EnvToken:             testToken,
		EnvAddress:           "192.0.2.10:8765",
		EnvAllowNonLoopback:  "1",
		EnvTrustedProxyCIDRs: "192.0.2.0/24",
	}), 2)
	if err != nil {
		t.Fatal(err)
	}
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "proxy-test"})
	h := NewHandler(cfg, server, nil)
	h.setReady(true)

	for _, test := range []struct {
		name       string
		remoteAddr string
		forwarded  string
		status     int
	}{
		{name: "untrusted direct peer", remoteAddr: "198.51.100.1:1234", status: http.StatusForbidden},
		{name: "trusted proxy", remoteAddr: "192.0.2.1:1234", forwarded: "198.51.100.2", status: http.StatusOK},
		{name: "malformed forwarding chain", remoteAddr: "192.0.2.1:1234", forwarded: "not-an-ip", status: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://192.0.2.10:8765/healthz", nil)
			req.Host = "192.0.2.10:8765"
			req.RemoteAddr = test.remoteAddr
			if test.forwarded != "" {
				req.Header.Set("X-Forwarded-For", test.forwarded)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != test.status {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.status, rec.Body.String())
			}
		})
	}
}

func TestHandlerRejectsEventReplayAndUnsupportedMethods(t *testing.T) {
	h := newTestHandler(t, 2)
	h.setReady(true)

	replay := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/mcp", nil)
	replay.Host = "127.0.0.1:8765"
	replay.Header.Set("Authorization", "Bearer "+testToken)
	replay.Header.Set("Last-Event-ID", "secret-event")
	replayRecorder := httptest.NewRecorder()
	h.ServeHTTP(replayRecorder, replay)
	if replayRecorder.Code != http.StatusBadRequest {
		t.Fatalf("replay status = %d, want 400", replayRecorder.Code)
	}

	options := httptest.NewRequest(http.MethodOptions, "http://127.0.0.1:8765/mcp", nil)
	options.Host = "127.0.0.1:8765"
	options.Header.Set("Authorization", "Bearer "+testToken)
	optionsRecorder := httptest.NewRecorder()
	h.ServeHTTP(optionsRecorder, options)
	if optionsRecorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("OPTIONS status = %d, want 405", optionsRecorder.Code)
	}
	if optionsRecorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("OPTIONS response enabled CORS")
	}
}

func TestHandlerAccessLogRedactsCredentialsAndRoutingState(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "log-test"})
	h := NewHandler(validTestConfig(2), server, logger)
	h.setReady(true)

	querySecret := "query-secret-value"
	sessionSecret := "session-secret-value"
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8765/mcp?secret="+querySecret, strings.NewReader(`{}`))
	req.Host = "127.0.0.1:8765"
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set(sessionIDHeader, sessionSecret)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	logged := logBuffer.String()
	for _, forbidden := range []string{testToken, querySecret, sessionSecret, "Authorization"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("access log leaked %q: %s", forbidden, logged)
		}
	}
	if !strings.Contains(logged, "route=mcp") || !strings.Contains(logged, "status=400") {
		t.Fatalf("access log missing safe fields: %s", logged)
	}
}

func TestHandlerRejectsOversizedBodyBeforeMCP(t *testing.T) {
	for _, test := range []struct {
		name          string
		contentLength int64
	}{
		{name: "known length", contentLength: 9},
		{name: "chunked length", contentLength: -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := validTestConfig(2)
			cfg.MaxBodyBytes = 8
			cfg.MaxInFlightBodyBytes = 8
			server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "body-limit-test"})
			h := NewHandler(cfg, server, nil)
			h.setReady(true)

			req := authenticatedPostRequest("http://127.0.0.1:8765/mcp", strings.Repeat("x", 9))
			req.ContentLength = test.contentLength
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413, body %q", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandlerRejectsAggregateBodySaturation(t *testing.T) {
	cfg := validTestConfig(3)
	cfg.MaxBodyBytes = 8
	cfg.MaxInFlightBodyBytes = 8
	cfg.MaxConcurrentRequests = 2
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "body-budget-test"})
	h := NewHandler(cfg, server, nil)
	h.setReady(true)

	started := make(chan struct{})
	release := make(chan struct{})
	h.mcpHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusAccepted)
	})

	firstDone := make(chan int, 1)
	go func() {
		req := authenticatedPostRequest("http://127.0.0.1:8765/mcp", strings.Repeat("x", 8))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		firstDone <- rec.Code
	}()
	<-started

	secondReq := authenticatedPostRequest("http://127.0.0.1:8765/mcp", "x")
	secondRec := httptest.NewRecorder()
	h.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusTooManyRequests || secondRec.Header().Get("Retry-After") == "" {
		t.Fatalf("body-budget response = %d headers=%v", secondRec.Code, secondRec.Header())
	}

	close(release)
	if status := <-firstDone; status != http.StatusAccepted {
		t.Fatalf("first request status = %d", status)
	}
}

func TestStreamableHTTPMatchesSharedServerAcrossAdapters(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve temporary root: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{
		Version:            "http-test",
		AllowedDirectories: []string{root},
		Config:             config.Load(),
		EnableClientRoots:  false,
		LifecycleContext:   ctx,
	})

	cfg := validTestConfig(4)
	unstarted := httptest.NewUnstartedServer(nil)
	cfg.AllowedHosts = map[string]struct{}{strings.ToLower(unstarted.Listener.Addr().String()): {}}
	h := NewHandler(cfg, server, nil)
	h.setReady(true)
	unstarted.Config.Handler = h
	unstarted.Start()
	t.Cleanup(func() {
		unstarted.CloseClientConnections()
		unstarted.Close()
	})

	first := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
	t.Cleanup(func() { _ = first.Close() })
	second := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
	t.Cleanup(func() { _ = second.Close() })
	direct := connectDirectClient(t, ctx, server)
	t.Cleanup(func() { _ = direct.Close() })
	sessions := map[string]*mcp.ClientSession{
		"http-first":  first,
		"http-second": second,
		"direct":      direct,
	}

	var expectedTools string
	var expectedPrompts string
	var expectedRoots string
	for name, session := range sessions {
		tools, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("%s list tools: %v", name, err)
		}
		if len(tools.Tools) != 25 {
			t.Fatalf("%s tool count = %d", name, len(tools.Tools))
		}
		serializedTools := marshalJSON(t, tools.Tools)
		if expectedTools == "" {
			expectedTools = serializedTools
		} else if serializedTools != expectedTools {
			t.Fatalf("%s tool metadata diverged", name)
		}

		prompts := make([]*mcp.Prompt, 0, 3)
		for prompt, promptErr := range session.Prompts(ctx, nil) {
			if promptErr != nil {
				t.Fatalf("%s list prompts: %v", name, promptErr)
			}
			prompts = append(prompts, prompt)
		}
		if len(prompts) != 3 {
			t.Fatalf("%s prompt count = %d", name, len(prompts))
		}
		serializedPrompts := marshalJSON(t, prompts)
		if expectedPrompts == "" {
			expectedPrompts = serializedPrompts
		} else if serializedPrompts != expectedPrompts {
			t.Fatalf("%s prompt metadata diverged", name)
		}

		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_allowed_directories"})
		if err != nil || result.IsError {
			t.Fatalf("%s roots result = %#v err=%v", name, result, err)
		}
		serializedRoots := marshalJSON(t, result.StructuredContent)
		if expectedRoots == "" {
			expectedRoots = serializedRoots
		} else if serializedRoots != expectedRoots {
			t.Fatalf("%s process roots diverged: %s != %s", name, serializedRoots, expectedRoots)
		}
	}

	notification, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		unstarted.URL+cfg.Path,
		strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/roots/list_changed","params":{}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	notification.Header.Set("Authorization", "Bearer "+testToken)
	notification.Header.Set("Content-Type", "application/json")
	notification.Header.Set("Accept", "application/json, text/event-stream")
	notification.Header.Set(sessionIDHeader, first.ID())
	if initialized := first.InitializeResult(); initialized != nil {
		notification.Header.Set(protocolVersionHeader, initialized.ProtocolVersion)
	}
	notification.Header.Set("Mcp-Method", "notifications/roots/list_changed")
	response, err := http.DefaultClient.Do(notification)
	if err != nil {
		t.Fatalf("send roots notification: %v", err)
	}
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("roots notification status = %d, body %q", response.StatusCode, readBody(t, response))
	}
	_ = response.Body.Close()
	rootsAfterNotification, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "list_allowed_directories"})
	if err != nil || rootsAfterNotification.IsError {
		t.Fatalf("roots after notification = %#v err=%v", rootsAfterNotification, err)
	}
	if got := marshalJSON(t, rootsAfterNotification.StructuredContent); got != expectedRoots {
		t.Fatalf("HTTP roots notification changed process roots: %s != %s", got, expectedRoots)
	}

	path := filepath.Join(root, "shared-cp1251.txt")
	writeResult, err := direct.CallTool(ctx, &mcp.CallToolParams{
		Name: "write_file",
		Arguments: map[string]any{
			"path":     path,
			"content":  "Привет",
			"encoding": "cp1251",
		},
	})
	if err != nil || writeResult.IsError {
		t.Fatalf("direct write result = %#v err=%v", writeResult, err)
	}
	var expectedRead string
	for name, session := range map[string]*mcp.ClientSession{"http": first, "direct": direct} {
		readResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "read_text_file",
			Arguments: map[string]any{
				"path":     path,
				"encoding": "cp1251",
			},
		})
		if err != nil || readResult.IsError {
			t.Fatalf("%s read result = %#v err=%v", name, readResult, err)
		}
		serializedRead := marshalJSON(t, readResult.StructuredContent)
		if expectedRead == "" {
			expectedRead = serializedRead
		} else if serializedRead != expectedRead {
			t.Fatalf("%s read result diverged: %s != %s", name, serializedRead, expectedRead)
		}
	}

	var expectedFingerprint string
	for name, session := range map[string]*mcp.ClientSession{"http": first, "direct": direct} {
		fingerprintResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "fingerprint_paths",
			Arguments: map[string]any{
				"paths":          []string{path},
				"includeEntries": true,
			},
		})
		if err != nil || fingerprintResult.IsError {
			t.Fatalf("%s fingerprint result = %#v err=%v", name, fingerprintResult, err)
		}
		serializedFingerprint := marshalJSON(t, fingerprintResult.StructuredContent)
		if expectedFingerprint == "" {
			expectedFingerprint = serializedFingerprint
		} else if serializedFingerprint != expectedFingerprint {
			t.Fatalf("%s fingerprint result diverged: %s != %s", name, serializedFingerprint, expectedFingerprint)
		}
	}
	var fingerprintOutput struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.Unmarshal([]byte(expectedFingerprint), &fingerprintOutput); err != nil {
		t.Fatal(err)
	}
	packageOriginal, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	packageManifest := map[string]any{
		"formatVersion":        "patch-package-v1",
		"fingerprintAlgorithm": "sha256",
		"fingerprintMode":      "content-v1",
		"targets": []map[string]any{{
			"path":                path,
			"expectedFingerprint": fingerprintOutput.Fingerprint,
			"encoding":            "cp1251",
			"edits":               []map[string]any{{"oldText": "Привет", "newText": "Здравствуйте"}},
		}},
	}
	var expectedInspect string
	for name, session := range map[string]*mcp.ClientSession{"http": first, "direct": direct} {
		inspectResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "patch_package",
			Arguments: map[string]any{"action": "inspect", "manifest": packageManifest},
		})
		if err != nil || inspectResult.IsError {
			t.Fatalf("%s patch package inspect result = %#v err=%v", name, inspectResult, err)
		}
		serialized := marshalJSON(t, inspectResult.StructuredContent)
		if expectedInspect == "" {
			expectedInspect = serialized
		} else if serialized != expectedInspect {
			t.Fatalf("%s patch package inspect diverged: %s != %s", name, serialized, expectedInspect)
		}
	}

	packageResult, err := first.CallTool(ctx, &mcp.CallToolParams{
		Name:      "patch_package",
		Arguments: map[string]any{"action": "dryRun", "manifest": packageManifest},
	})
	if err != nil || packageResult.IsError {
		t.Fatalf("HTTP patch package dryRun result = %#v err=%v", packageResult, err)
	}
	var packagePreview struct {
		PreviewID string `json:"previewId"`
		Results   []struct {
			ResultFingerprint string `json:"resultFingerprint"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(marshalJSON(t, packageResult.StructuredContent)), &packagePreview); err != nil {
		t.Fatal(err)
	}
	if len(packagePreview.PreviewID) != 64 || len(packagePreview.Results) != 1 || len(packagePreview.Results[0].ResultFingerprint) != 64 {
		t.Fatalf("unexpected package preview: %#v", packagePreview)
	}
	readAfterPackage, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readAfterPackage, packageOriginal) {
		t.Fatal("patch package dryRun changed the CP1251 fixture")
	}

	packageApply, err := direct.CallTool(ctx, &mcp.CallToolParams{
		Name: "patch_package",
		Arguments: map[string]any{
			"action":    "apply",
			"previewId": packagePreview.PreviewID,
		},
	})
	if err != nil || packageApply.IsError {
		t.Fatalf("direct patch package apply result = %#v err=%v", packageApply, err)
	}
	var packageApplyOutput struct {
		Applied   bool   `json:"applied"`
		PreviewID string `json:"previewId"`
	}
	if err := json.Unmarshal([]byte(marshalJSON(t, packageApply.StructuredContent)), &packageApplyOutput); err != nil {
		t.Fatal(err)
	}
	if !packageApplyOutput.Applied || packageApplyOutput.PreviewID != "" {
		t.Fatalf("unexpected package apply output: %#v", packageApplyOutput)
	}
	packageReplay, err := second.CallTool(ctx, &mcp.CallToolParams{
		Name: "patch_package",
		Arguments: map[string]any{
			"action":    "apply",
			"previewId": packagePreview.PreviewID,
		},
	})
	if err != nil || !packageReplay.IsError || packageReplay.Meta[handler.ErrorCodeMetaKey] != handler.ErrCodeConflict {
		t.Fatalf("HTTP package replay result = %#v err=%v", packageReplay, err)
	}

	verifyManifest := map[string]any{
		"formatVersion":        "patch-package-v1",
		"fingerprintAlgorithm": "sha256",
		"fingerprintMode":      "content-v1",
		"targets": []map[string]any{{
			"path":                      path,
			"expectedFingerprint":       fingerprintOutput.Fingerprint,
			"expectedResultFingerprint": packagePreview.Results[0].ResultFingerprint,
			"encoding":                  "cp1251",
			"edits":                     []map[string]any{{"oldText": "Привет", "newText": "Здравствуйте"}},
		}},
	}
	var expectedVerify string
	for name, session := range map[string]*mcp.ClientSession{"http": second, "direct": direct} {
		verifyResult, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "patch_package",
			Arguments: map[string]any{"action": "verify", "manifest": verifyManifest},
		})
		if err != nil || verifyResult.IsError {
			t.Fatalf("%s patch package verify result = %#v err=%v", name, verifyResult, err)
		}
		serialized := marshalJSON(t, verifyResult.StructuredContent)
		if expectedVerify == "" {
			expectedVerify = serialized
		} else if serialized != expectedVerify {
			t.Fatalf("%s patch package verify diverged: %s != %s", name, serialized, expectedVerify)
		}
	}

	previewPath := filepath.Join(root, "preview.txt")
	if err := os.WriteFile(previewPath, []byte("alpha"), 0644); err != nil {
		t.Fatal(err)
	}
	previewResult, err := first.CallTool(ctx, &mcp.CallToolParams{
		Name: "edit_file",
		Arguments: map[string]any{
			"action": "preview",
			"path":   previewPath,
			"edits":  []map[string]any{{"oldText": "alpha", "newText": "omega"}},
		},
	})
	if err != nil || previewResult.IsError {
		t.Fatalf("HTTP edit preview result = %#v err=%v", previewResult, err)
	}
	var previewOutput struct {
		PreviewID string `json:"previewId"`
		Changed   bool   `json:"changed"`
	}
	if err := json.Unmarshal([]byte(marshalJSON(t, previewResult.StructuredContent)), &previewOutput); err != nil {
		t.Fatal(err)
	}
	if len(previewOutput.PreviewID) != 64 || !previewOutput.Changed {
		t.Fatalf("unexpected HTTP edit preview output: %#v", previewOutput)
	}
	if data, err := os.ReadFile(previewPath); err != nil || string(data) != "alpha" {
		t.Fatalf("HTTP preview changed target: %q err=%v", data, err)
	}

	applyResult, err := direct.CallTool(ctx, &mcp.CallToolParams{
		Name: "edit_file",
		Arguments: map[string]any{
			"action":    "apply",
			"previewId": previewOutput.PreviewID,
		},
	})
	if err != nil || applyResult.IsError {
		t.Fatalf("direct edit apply result = %#v err=%v", applyResult, err)
	}
	var applyOutput struct {
		PreviewID string `json:"previewId"`
		Applied   bool   `json:"applied"`
	}
	if err := json.Unmarshal([]byte(marshalJSON(t, applyResult.StructuredContent)), &applyOutput); err != nil {
		t.Fatal(err)
	}
	if !applyOutput.Applied || applyOutput.PreviewID != "" {
		t.Fatalf("unexpected direct edit apply output: %#v", applyOutput)
	}
	if data, err := os.ReadFile(previewPath); err != nil || string(data) != "omega" {
		t.Fatalf("direct apply content: %q err=%v", data, err)
	}
	replayResult, err := second.CallTool(ctx, &mcp.CallToolParams{
		Name: "edit_file",
		Arguments: map[string]any{
			"action":    "apply",
			"previewId": previewOutput.PreviewID,
		},
	})
	if err != nil || !replayResult.IsError || replayResult.Meta[handler.ErrorCodeMetaKey] != handler.ErrCodeConflict {
		t.Fatalf("HTTP replay result = %#v err=%v", replayResult, err)
	}

	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	var expectedErrorCode any
	var expectedErrorText string
	for name, session := range map[string]*mcp.ClientSession{"http": second, "direct": direct} {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "get_file_info",
			Arguments: map[string]any{"path": outside},
		})
		if err != nil {
			t.Fatalf("%s access error: %v", name, err)
		}
		if !result.IsError {
			t.Fatalf("%s access outside roots unexpectedly succeeded", name)
		}
		errorCode := result.Meta["errorCode"]
		errorText := firstTextContent(result)
		if expectedErrorCode == nil {
			expectedErrorCode = errorCode
			expectedErrorText = errorText
		} else if errorCode != expectedErrorCode || errorText != expectedErrorText {
			t.Fatalf("%s error diverged: code=%v text=%q, want code=%v text=%q", name, errorCode, errorText, expectedErrorCode, expectedErrorText)
		}
	}
}

func TestHTTPExecutionRequiresExplicitServerPolicy(t *testing.T) {
	t.Setenv("MCP_ENABLE_EXECUTION", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, test := range []struct {
		name     string
		policy   handler.ExecutionPolicy
		wantText string
	}{
		{name: "transport gate closed", policy: handler.ExecutionPolicy{}, wantText: "shell is disabled"},
		{name: "transport gate open", policy: handler.ExecutionPolicy{AllowShell: true}, wantText: "command is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{
				Version:          "execution-policy-test",
				ExecutionPolicy:  &test.policy,
				LifecycleContext: ctx,
			})
			cfg := validTestConfig(2)
			unstarted := httptest.NewUnstartedServer(nil)
			cfg.AllowedHosts = map[string]struct{}{strings.ToLower(unstarted.Listener.Addr().String()): {}}
			h := NewHandler(cfg, server, nil)
			h.setReady(true)
			unstarted.Config.Handler = h
			unstarted.Start()
			defer unstarted.Close()

			session := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
			defer session.Close()
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "shell",
				Arguments: map[string]any{"command": ""},
			})
			if err != nil {
				t.Fatalf("call shell: %v", err)
			}
			if !result.IsError || !strings.Contains(firstTextContent(result), test.wantText) {
				t.Fatalf("shell result = %#v, want text containing %q", result, test.wantText)
			}
		})
	}
}

func TestConcurrentRequestLimitRejectsSaturation(t *testing.T) {
	cfg := validTestConfig(2)
	cfg.MaxConcurrentRequests = 1
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "concurrency-test"})
	h := NewHandler(cfg, server, nil)
	h.setReady(true)

	started := make(chan struct{})
	release := make(chan struct{})
	h.mcpHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusAccepted)
	})

	firstDone := make(chan int, 1)
	go func() {
		req := authenticatedPostRequest("http://127.0.0.1:8765/mcp", `{}`)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		firstDone <- rec.Code
	}()
	<-started

	secondReq := authenticatedPostRequest("http://127.0.0.1:8765/mcp", `{}`)
	secondRec := httptest.NewRecorder()
	h.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusTooManyRequests || secondRec.Header().Get("Retry-After") == "" {
		t.Fatalf("saturated response = %d headers=%v", secondRec.Code, secondRec.Header())
	}

	close(release)
	if status := <-firstDone; status != http.StatusAccepted {
		t.Fatalf("first request status = %d", status)
	}
}

func TestHTTPDisconnectCancelsToolContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	started := make(chan struct{})
	cancelled := make(chan struct{})

	server := mcp.NewServer(&mcp.Implementation{Name: "cancel-test", Version: "test"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "block"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, struct{}{}, ctx.Err()
	})

	cfg := validTestConfig(2)
	unstarted := httptest.NewUnstartedServer(nil)
	cfg.AllowedHosts = map[string]struct{}{strings.ToLower(unstarted.Listener.Addr().String()): {}}
	h := NewHandler(cfg, server, nil)
	h.setReady(true)
	unstarted.Config.Handler = h
	unstarted.Start()
	defer unstarted.Close()

	session := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
	defer session.Close()
	callCtx, cancelCall := context.WithCancel(ctx)
	callDone := make(chan error, 1)
	go func() {
		_, err := session.CallTool(callCtx, &mcp.CallToolParams{Name: "block"})
		callDone <- err
	}()
	<-started
	cancelCall()
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("tool context was not cancelled after HTTP call cancellation")
	}
	select {
	case <-callDone:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled HTTP call did not return")
	}
}

func TestSessionIDsAreUniqueVisibleASCII(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "session-id-test", LifecycleContext: ctx})
	cfg := validTestConfig(16)
	unstarted := httptest.NewUnstartedServer(nil)
	cfg.AllowedHosts = map[string]struct{}{strings.ToLower(unstarted.Listener.Addr().String()): {}}
	h := NewHandler(cfg, server, nil)
	h.limiter = newPeerLimiter(1000, 1000, defaultRatePeers, defaultRatePeerIdle)
	h.setReady(true)
	unstarted.Config.Handler = h
	unstarted.Start()
	defer unstarted.Close()

	seen := make(map[string]struct{})
	for index := 0; index < 12; index++ {
		session := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
		sessionID := session.ID()
		if sessionID == "" {
			t.Fatal("empty session ID")
		}
		for _, current := range sessionID {
			if current < 0x21 || current > 0x7e {
				t.Fatalf("session ID contains non-visible ASCII: %q", sessionID)
			}
		}
		if _, duplicate := seen[sessionID]; duplicate {
			t.Fatalf("duplicate session ID %q", sessionID)
		}
		seen[sessionID] = struct{}{}
		_ = session.Close()
	}
}

func TestSSEOnlySessionReleasesCapacityAfterIdleTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "sse-expiry-test", LifecycleContext: ctx})
	cfg := validTestConfig(1)
	cfg.SessionTimeout = 40 * time.Millisecond
	unstarted := httptest.NewUnstartedServer(nil)
	cfg.AllowedHosts = map[string]struct{}{strings.ToLower(unstarted.Listener.Addr().String()): {}}
	h := NewHandler(cfg, server, nil)
	h.setReady(true)
	unstarted.Config.Handler = h
	unstarted.Start()
	defer unstarted.Close()

	first := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
	defer first.Close()
	time.Sleep(cfg.SessionTimeout + sessionTrackerGrace + 150*time.Millisecond)
	second := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
	_ = second.Close()
}

func TestSessionLimitReleasesAfterDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "limit-test", LifecycleContext: ctx})
	cfg := validTestConfig(1)
	unstarted := httptest.NewUnstartedServer(nil)
	cfg.AllowedHosts = map[string]struct{}{strings.ToLower(unstarted.Listener.Addr().String()): {}}
	h := NewHandler(cfg, server, nil)
	h.setReady(true)
	unstarted.Config.Handler = h
	unstarted.Start()
	defer unstarted.Close()

	first := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
	if _, err := tryConnectHTTPClient(ctx, unstarted.URL+cfg.Path); err == nil {
		t.Fatal("second session exceeded configured limit")
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}
	second := connectHTTPClient(t, ctx, unstarted.URL+cfg.Path)
	_ = second.Close()
}

func newTestHandler(t *testing.T, maxSessions int) *Handler {
	t.Helper()
	server := filetoolsserver.BuildServer(filetoolsserver.ServerOptions{Version: "middleware-test"})
	return NewHandler(validTestConfig(maxSessions), server, nil)
}

func validTestConfig(maxSessions int) Config {
	cfg, err := LoadConfig(environment(map[string]string{EnvToken: testToken}), maxSessions)
	if err != nil {
		panic(err)
	}
	return cfg
}

func authenticatedPostRequest(url, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Host = "127.0.0.1:8765"
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	return req
}

func connectDirectClient(t *testing.T, ctx context.Context, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect direct server: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "direct-test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect direct client: %v", err)
	}
	return clientSession
}

func connectHTTPClient(t *testing.T, ctx context.Context, endpoint string) *mcp.ClientSession {
	t.Helper()
	session, err := tryConnectHTTPClient(ctx, endpoint)
	if err != nil {
		t.Fatalf("connect HTTP client: %v", err)
	}
	return session
}

func tryConnectHTTPClient(ctx context.Context, endpoint string) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "http-test", Version: "test"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint: endpoint,
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("Authorization", "Bearer "+testToken)
			return http.DefaultTransport.RoundTrip(req)
		})},
		MaxRetries: -1,
	}
	return client.Connect(ctx, transport, nil)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func marshalJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return string(data)
}

func firstTextContent(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	text, _ := result.Content[0].(*mcp.TextContent)
	if text == nil {
		return ""
	}
	return text.Text
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

var testToken = strings.Repeat("test-token-", 4)
