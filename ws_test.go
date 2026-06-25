package ctgexchange

import "testing"

// WebSocket frame parsing — only data frames reach the consumer.

func TestParseFrameSnapshot(t *testing.T) {
	msg, ok := parseFrame([]byte(
		`{"type":"snapshot","channel":"orderbook","symbol":"CTGUSDT","data":{"x":1}}`))
	if !ok {
		t.Fatal("want ok")
	}
	if msg.Type != "snapshot" || msg.Channel != "orderbook" || msg.Symbol != "CTGUSDT" {
		t.Errorf("unexpected: %+v", msg)
	}
	if string(msg.Data) != `{"x":1}` {
		t.Errorf("data: got %s", msg.Data)
	}
}

func TestParseFrameUpdate(t *testing.T) {
	if _, ok := parseFrame([]byte(`{"type":"update","channel":"ticker","data":{}}`)); !ok {
		t.Error("want ok for update frame")
	}
}

func TestParseFrameSkipsControlAndJunk(t *testing.T) {
	for _, raw := range []string{
		`{"type":"subscribed","channels":["x"]}`,
		`{"op":"auth","success":true}`,
		`not json`,
		`"a bare string"`,
		`[1,2,3]`,
	} {
		if _, ok := parseFrame([]byte(raw)); ok {
			t.Errorf("should have skipped: %s", raw)
		}
	}
}

func TestNewUserStreamRequiresCredentials(t *testing.T) {
	if _, err := NewUserStream("", "", StreamConfig{}); err == nil {
		t.Error("want an error for missing credentials")
	}
}
