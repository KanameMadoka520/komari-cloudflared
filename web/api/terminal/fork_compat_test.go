package terminal

import (
	"testing"
	"time"
)

func TestAgentTerminationIncludesLegacyClose(t *testing.T) {
	conn, client := newControlPair(t)
	notifyAgentTerminate(conn)
	if frame := readControlFrame(t, client); frame.Type != controlTypeTerminate {
		t.Fatalf("first frame: %q", frame.Type)
	}
	if frame := readControlFrame(t, client); frame.Type != "close" {
		t.Fatalf("legacy agent would retain its PTY: %q", frame.Type)
	}
}

func TestReplacedConnectionsCannotTerminateSession(t *testing.T) {
	resetTerminalSessions(t)
	oldBrowser, _ := newControlPair(t)
	browser, _ := newControlPair(t)
	oldAgent, _ := newControlPair(t)
	agent, _ := newControlPair(t)
	session := newSession(t, "replaced", "owner", "client")
	session.Browser, session.Agent = browser, agent
	closeSessionMatching("replaced", oldBrowser, nil)
	closeSessionMatching("replaced", nil, oldAgent)
	TerminalSessionsMutex.Lock()
	present := TerminalSessions["replaced"] == session
	TerminalSessionsMutex.Unlock()
	if !present {
		t.Fatal("stale connection terminated the replacement session")
	}
	closeSession("replaced")
}

func TestStaleForwarderReturns(t *testing.T) {
	resetTerminalSessions(t)
	browser, _ := newControlPair(t)
	agent, _ := newControlPair(t)
	done := make(chan struct{})
	go func() { ForwardTerminal("missing", browser, agent); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stale forwarder leaked waiting on a nil channel")
	}
}
