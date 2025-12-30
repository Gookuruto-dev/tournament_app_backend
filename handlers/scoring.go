package handlers

import (
	"bbx_tournament/db"
	"bbx_tournament/models"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ScoreRequest represents the payload for a score update
type ScoreRequest struct {
	WinnerID uint   `json:"winner_id"` // ID of the player who won the round
	WinType  string `json:"win_type"`  // Spin, Over, Burst, Out, Xtreme
	// Correction: "Out" and "Over" might be same/similar in some contexts but rules say:
	// Over Finish (2), Out Finish (2), Burst (2), Spin (1), Xtreme (3)
}

// Points map based on rules
var pointsMap = map[string]int{
	"Spin":   1,
	"Over":   2,
	"Burst":  2,
	"Out":    2,
	"Xtreme": 3,
}

// UpdateMatchScore handles round updates
func UpdateMatchScore(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "id")
	matchID, _ := strconv.Atoi(matchIDStr)

	var req ScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	points, ok := pointsMap[req.WinType]
	if !ok {
		http.Error(w, "Invalid WinType", http.StatusBadRequest)
		return
	}

	var m models.Match
	// Preload to return full object if needed, or just update
	if err := db.DB.Preload("Player1").Preload("Player2").First(&m, matchID).Error; err != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	if m.WinnerID != nil {
		http.Error(w, "Match already finished", http.StatusBadRequest)
		return
	}

	if m.Player1ID != nil && req.WinnerID == *m.Player1ID {
		m.ScoreP1 += points
	} else if m.Player2ID != nil && req.WinnerID == *m.Player2ID {
		m.ScoreP2 += points
	} else {
		http.Error(w, "Invalid WinnerID", http.StatusBadRequest)
		return
	}

	// Check Win Condition (Group Stage: 7pts)
	// TODO: Dynamic limit based on phase (Group=7, Bracket=10)
	winLimit := 7
	if m.Phase == "Bracket" {
		winLimit = 10
	}

	if m.ScoreP1 >= winLimit {
		m.WinnerID = m.Player1ID
	} else if m.ScoreP2 >= winLimit {
		m.WinnerID = m.Player2ID
	}

	if err := db.DB.Save(&m).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update Stats if someone won
	if m.WinnerID != nil {
		updateWinnerStats(m.TournamentID, *m.WinnerID, req.WinType)
		// Automatic progression to next match
		if m.NextMatchID != nil {
			var nm models.Match
			if err := db.DB.First(&nm, *m.NextMatchID).Error; err == nil {
				if m.NextMatchSlot == 1 {
					nm.Player1ID = m.WinnerID
				} else if m.NextMatchSlot == 2 {
					nm.Player2ID = m.WinnerID
				}
				db.DB.Save(&nm)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

// ResetMatch resets the scores and winner of a match
func ResetMatch(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "id")
	matchID, _ := strconv.Atoi(matchIDStr)

	var match models.Match
	if result := db.DB.Preload("Player1").Preload("Player2").First(&match, matchID); result.Error != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	match.ScoreP1 = 0
	match.ScoreP2 = 0
	match.WinnerID = nil

	if err := db.DB.Save(&match).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(match)
}

// ManualScoreRequest
type ManualScoreRequest struct {
	ScoreP1 int `json:"score_p1"`
	ScoreP2 int `json:"score_p2"`
}

// ManualMatchScore sets the score directly
func ManualMatchScore(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "id")
	matchID, _ := strconv.Atoi(matchIDStr)

	var req ManualScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var match models.Match
	if result := db.DB.Preload("Player1").Preload("Player2").First(&match, matchID); result.Error != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	match.ScoreP1 = req.ScoreP1
	match.ScoreP2 = req.ScoreP2

	// Check for Winner override or clear
	winLimit := 7
	if match.Phase == "Bracket" {
		winLimit = 10
	}

	if match.ScoreP1 >= winLimit {
		match.WinnerID = match.Player1ID
	} else if match.ScoreP2 >= winLimit {
		match.WinnerID = match.Player2ID
	} else {
		match.WinnerID = nil // Clear winner if score drops below limit
	}

	if err := db.DB.Save(&match).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Automatic progression to next match
	if match.WinnerID != nil && match.NextMatchID != nil {
		var nm models.Match
		if err := db.DB.First(&nm, *match.NextMatchID).Error; err == nil {
			if match.NextMatchSlot == 1 {
				nm.Player1ID = match.WinnerID
			} else if match.NextMatchSlot == 2 {
				nm.Player2ID = match.WinnerID
			}
			db.DB.Save(&nm)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(match)
}

func updateWinnerStats(tournamentID, participantID uint, winType string) {
	var tp models.TournamentParticipant
	if err := db.DB.Where("tournament_id = ? AND participant_id = ?", tournamentID, participantID).First(&tp).Error; err != nil {
		return
	}

	tp.Wins++

	switch winType {
	case "Spin":
		tp.SpinFinishes++
	case "Burst":
		tp.BurstFinishes++
	case "Over", "Out":
		tp.OverFinishes++
	case "Xtreme":
		tp.XtremeFinishes++
	}

	tp.Points += 3
	db.DB.Save(&tp)
}
