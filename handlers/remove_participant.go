package handlers

import (
	"bbx_tournament/db"
	"bbx_tournament/models"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func RemoveParticipantFromTournament(w http.ResponseWriter, r *http.Request) {
	tourIDStr := chi.URLParam(r, "id")
	participantIDStr := chi.URLParam(r, "participantId")

	tourID, _ := strconv.Atoi(tourIDStr)
	participantID, _ := strconv.Atoi(participantIDStr)

	var t models.Tournament
	if err := db.DB.First(&t, tourID).Error; err != nil {
		http.Error(w, "Tournament not found", http.StatusNotFound)
		return
	}

	// Only allow removal if tournament hasn't started
	if t.Status != "Created" {
		http.Error(w, "Cannot remove participant after tournament has started", http.StatusBadRequest)
		return
	}

	// Delete the tournament participant record
	if err := db.DB.Where("tournament_id = ? AND participant_id = ?", tourID, participantID).
		Delete(&models.TournamentParticipant{}).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "removed"}`))
}
