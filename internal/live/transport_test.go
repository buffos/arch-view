package live

import (
	"context"
	"testing"
)

func newTransportTestSession(t *testing.T, sessionID string) *LiveSession {
	t.Helper()
	root := t.TempDir()
	config := testLiveConfig(sessionID)
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{
		Scanner:       StaticScanner{Result: ScanResult{Model: testModel()}},
		Fingerprinter: StaticFingerprinter{Value: testInput("transport")},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	return session
}
