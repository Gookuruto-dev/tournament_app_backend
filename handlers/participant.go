package handlers

import (
	"bbx_tournament/db"
	"bbx_tournament/models"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GetParticipants returns all non-archived participants
func GetParticipants(w http.ResponseWriter, r *http.Request) {
	var participants []models.Participant
	// Allow ?include_archived=true for the archives page
	query := db.DB.Where("is_archived = ?", false)
	if r.URL.Query().Get("include_archived") == "true" {
		query = db.DB
	}

	if result := query.Find(&participants); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}

// CreateParticipant adds a new participant
func CreateParticipant(w http.ResponseWriter, r *http.Request) {
	var participant models.Participant
	if err := json.NewDecoder(r.Body).Decode(&participant); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if result := db.DB.Create(&participant); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(participant)
}

// ArchiveParticipant soft-deletes a participant
func ArchiveParticipant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.Model(&models.Participant{}).Where("id = ?", idStr).Update("is_archived", true).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "archived"}`))
}

type ParticipantStats struct {
	ParticipantID     uint   `json:"participant_id"`
	Nickname          string `json:"nickname"`
	TotalWins         int    `json:"total_wins"`
	TotalPoints       int    `json:"total_points"`
	TotalLeaguePoints int    `json:"total_league_points"` // The new custom points
	TotalSpin         int    `json:"total_spin"`
	TotalBurst        int    `json:"total_burst"`
	TotalOver         int    `json:"total_over"`
	TotalXtreme       int    `json:"total_xtreme"`
	TournamentsPlayed int    `json:"tournaments_played"`
}

// GetLeagueStats aggregates stats across all tournaments
func GetLeagueStats(w http.ResponseWriter, r *http.Request) {
	var results []ParticipantStats

	err := db.DB.Model(&models.TournamentParticipant{}).
		Select("participant_id, participants.nickname, sum(wins) as total_wins, sum(points) as total_points, sum(league_points) as total_league_points, sum(spin_finishes) as total_spin, sum(burst_finishes) as total_burst, sum(over_finishes) as total_over, sum(xtreme_finishes) as total_xtreme, count(tournament_id) as tournaments_played").
		Joins("join participants on participants.id = tournament_participants.participant_id").
		Where("participants.is_archived = ?", false).
		Group("participant_id, participants.nickname").
		Order("total_league_points DESC, total_wins DESC").
		Scan(&results).Error

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
