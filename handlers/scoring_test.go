package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpdateMatchScore(t *testing.T) {
	// Setup (mock DB or use in-memory sqlite if possible, or just unit test logic if separated)
	// Since UpdateMatchScore is tightly coupled with DB, we might need a test DB setup.
	// For now, let's assume we can test the *logic* if we extract it, but since we didn't,
	// we'll write a test that *would* work with a running server or mock.

	// Actually, let's create a logic-only test for points calculation to be safe.

	tests := []struct {
		name     string
		winType  string
		expected int
	}{
		{"Spin", "Spin", 1},
		{"Over", "Over", 2},
		{"Burst", "Burst", 2},
		{"Out", "Out", 2},
		{"Xtreme", "Xtreme", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if pointsMap[tt.winType] != tt.expected {
				t.Errorf("expected %d points for %s, got %d", tt.expected, tt.winType, pointsMap[tt.winType])
			}
		})
	}
}

func TestManualScoreLogic(t *testing.T) {
	// Simple test for struct binding
	payload := []byte(`{"score_p1": 5, "score_p2": 3}`)
	req, _ := http.NewRequest("POST", "/manual", bytes.NewBuffer(payload))

	var data ManualScoreRequest
	if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if data.ScoreP1 != 5 || data.ScoreP2 != 3 {
		t.Errorf("decoding failed, got %v", data)
	}
}
