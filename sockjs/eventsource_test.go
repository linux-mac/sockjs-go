package sockjs

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_EventSource(t *testing.T) {
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/eventsource", nil)
	h := newTestHandler()
	go func() {
		h.sessionsMux.Lock()
		defer h.sessionsMux.Unlock()
		sess := h.sessions["session"]
		sess.Lock()
		defer sess.Unlock()
		recv := sess.recv
		recv.close()
	}()
	h.eventSource(rw, req)
	contentType := rw.Header().Get("content-type")
	expected := "text/event-stream; charset=UTF-8"
	if contentType != expected {
		t.Errorf("Unexpected content type, got '%s', extected '%s'", contentType, expected)
	}
	if rw.Code != http.StatusOK {
		t.Errorf("Unexpected response code, got '%d', expected '%d'", rw.Code, http.StatusOK)
	}

	if rw.Body.String() != "\r\ndata: o\r\n\r\n" {
		t.Errorf("Event stream prelude, got '%s'", rw.Body)
	}
}

func TestHandler_EventSourceMultipleConnections(t *testing.T) {
	h := newTestHandler()
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/sess/eventsource", nil)
	var doneCh = make(chan struct{})
	go func() {
		rw := httptest.NewRecorder()
		h.eventSource(rw, req)
		if rw.Body.String() != "\r\ndata: c[2010,\"Another connection still open\"]\r\n\r\n" {
			t.Errorf("wrong, got '%v'", rw.Body)
		}
		close(doneCh)

	}()
	h.eventSource(rw, req)
	<-doneCh
}

func TestHandler_EventSourceConnectionInterrupted(t *testing.T) {
	h := newTestHandler()
	sess := newTestSession()
	sess.state = sessionActive
	h.sessions["session"] = sess
	req, _ := http.NewRequest("POST", "/server/session/eventsource", nil)
	rw := newClosableRecorder()
	close(rw.closeNotifCh)
	h.eventSource(rw, req)
	sess.Lock()
	if sess.state != sessionClosed {
		t.Errorf("Session should be closed")
	}
}
