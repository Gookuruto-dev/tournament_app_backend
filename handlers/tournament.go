package handlers

import (
	"bbx_tournament/db"
	"bbx_tournament/models"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// GetTournaments list non-archived tournaments
func GetTournaments(w http.ResponseWriter, r *http.Request) {
	var tournaments []models.Tournament
	query := db.DB.Where("is_archived = ?", false)
	if r.URL.Query().Get("include_archived") == "true" {
		query = db.DB
	}

	if result := query.Find(&tournaments); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tournaments)
}

// CreateTournament creates a new tournament
func CreateTournament(w http.ResponseWriter, r *http.Request) {
	var t models.Tournament
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults
	if t.Date.IsZero() {
		t.Date = time.Now()
	}
	t.Status = "Created"

	if result := db.DB.Create(&t); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// GetTournamentDetails gets a single tournament with matches
func GetTournamentDetails(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var t models.Tournament
	// Preload everything we might need
	if result := db.DB.Preload("Matches.Player1").Preload("Matches.Player2").Preload("TournamentParticipants.Participant").First(&t, id); result.Error != nil {
		http.Error(w, "Tournament not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// AddParticipantToTournament registers a user for a tournament
func AddParticipantToTournament(w http.ResponseWriter, r *http.Request) {
	tourIDStr := chi.URLParam(r, "id")
	tourID, _ := strconv.Atoi(tourIDStr) // handle err

	var data struct {
		ParticipantID uint `json:"participant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Transaction to safeguard
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var t models.Tournament
		if err := tx.First(&t, tourID).Error; err != nil {
			return err
		}

		// Check if already joined (using new association)
		var existing models.TournamentParticipant
		if err := tx.Where("tournament_id = ? AND participant_id = ?", tourID, data.ParticipantID).First(&existing).Error; err == nil {
			return nil // Already joined, idempotent
		}

		tp := models.TournamentParticipant{
			TournamentID:  uint(tourID),
			ParticipantID: data.ParticipantID,
			Group:         "", // Added later
		}
		if err := tx.Create(&tp).Error; err != nil {
			return err
		}

		// Maintain legacy compatibility if needed, or rely on TournamentParticipants
		// Only adding to new table is fine if we update queries.
		// BUT: GenerateRoundRobin originally used t.Participants.
		// Let's add to old relation too just in case other parts break, or drop usage.
		// Simplest: We are rewriting logic, so rely on models.TournamentParticipant.
		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
}

// GenerateGroups assigns participants to groups
func GenerateGroups(w http.ResponseWriter, r *http.Request) {
	tourIDStr := chi.URLParam(r, "id")
	tourID, _ := strconv.Atoi(tourIDStr)

	var t models.Tournament
	if result := db.DB.Preload("TournamentParticipants").First(&t, tourID); result.Error != nil {
		http.Error(w, "Tournament not found", http.StatusNotFound)
		return
	}

	if t.Status != "Created" && t.Status != "GroupsGenerated" {
		http.Error(w, "Tournament already started or finished", http.StatusBadRequest)
		return
	}

	participants := t.TournamentParticipants
	n := len(participants)
	if n < 2 {
		http.Error(w, "Not enough participants", http.StatusBadRequest)
		return
	}

	// Logic: Target ~10 players per group.
	numGroups := 1
	if n > 10 {
		numGroups = (n + 9) / 10
	}

	groups := make([][]int, numGroups) // Indices of participants
	for i := range participants {
		groups[i%numGroups] = append(groups[i%numGroups], i)
	}

	groupNames := []string{"A", "B", "C", "D", "E", "F", "G", "H"}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		for gIdx, indices := range groups {
			groupLabel := groupNames[gIdx%len(groupNames)]
			if n > 10 {
				groupLabel = "Group " + groupLabel
			}
			for _, pIdx := range indices {
				if err := tx.Model(&participants[pIdx]).Update("group", groupLabel).Error; err != nil {
					return err
				}
			}
		}
		t.Status = "GroupsGenerated"
		return tx.Save(&t).Error
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Refetch with preloads
	db.DB.Preload("TournamentParticipants.Participant").First(&t, tourID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

// GenerateMatches creates matches based on assigned groups
func GenerateMatches(w http.ResponseWriter, r *http.Request) {
	tourIDStr := chi.URLParam(r, "id")
	tourID, _ := strconv.Atoi(tourIDStr)

	var t models.Tournament
	// Preload nested to get Participant details
	if result := db.DB.Preload("TournamentParticipants.Participant").First(&t, tourID); result.Error != nil {
		http.Error(w, "Tournament not found", http.StatusNotFound)
		return
	}

	if t.Status != "GroupsGenerated" {
		http.Error(w, "Groups must be generated first", http.StatusBadRequest)
		return
	}

	matches := generateMatchesFromGroups(t.ID, t.TournamentParticipants)

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&matches).Error; err != nil {
			return err
		}
		// Award Participation Points (5)
		var participants []models.TournamentParticipant
		if err := tx.Where("tournament_id = ?", tourID).Find(&participants).Error; err != nil {
			return err
		}
		for i := range participants {
			participants[i].AwardParticipation()
			if err := tx.Save(&participants[i]).Error; err != nil {
				return err
			}
		}
		t.Status = "InProgress"
		return tx.Save(&t).Error
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Refetch with preloads for proper response
	db.DB.Preload("Matches.Player1").Preload("Matches.Player2").Preload("TournamentParticipants.Participant").First(&t, tourID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

// Deprecated: StartTournament (Renaming or removing old logic)
// Kept for interface compatibility but shouldn't be used if we switch frontend.
// BUT: User might still press "Start" if we don't update frontend immediately.
// Let's redirect StartTournament to do both if strictness not required, or return error.
func StartTournament(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Use Generate Groups then Generate Matches", http.StatusBadRequest)
}

// AdvanceTournamentPhase checks if the current phase is complete and advances the tournament
func AdvanceTournamentPhase(w http.ResponseWriter, r *http.Request) {
	tourIDStr := chi.URLParam(r, "id")
	tourID, _ := strconv.Atoi(tourIDStr)

	var t models.Tournament
	// Preload everything needed
	if result := db.DB.Preload("Matches").Preload("TournamentParticipants.Participant").First(&t, tourID); result.Error != nil {
		http.Error(w, "Tournament not found", http.StatusNotFound)
		return
	}

	// 1. Check if all matches in the current state are finished
	allFinished := true
	for _, m := range t.Matches {
		if m.WinnerID == nil {
			allFinished = false
			break
		}
	}

	if !allFinished {
		http.Error(w, "Current phase matches are not all finished", http.StatusBadRequest)
		return
	}

	// 2. Logic based on current status
	switch t.Status {
	case "InProgress": // Transitioning from Group Stage to Bracket
		// Group participants by group
		grouped := make(map[string][]models.TournamentParticipant)
		for _, tp := range t.TournamentParticipants {
			if tp.Group != "" {
				grouped[tp.Group] = append(grouped[tp.Group], tp)
			}
		}

		var qualifiedIDs []uint
		// Sort each group and take top 4
		for _, members := range grouped {
			sort.Slice(members, func(i, j int) bool {
				if members[i].Points != members[j].Points {
					return members[i].Points > members[j].Points
				}
				// Tie-breaker: Wins
				return members[i].Wins > members[j].Wins
			})

			// Take top 4 (or less if group is smaller)
			limit := 4
			if len(members) < limit {
				limit = len(members)
			}
			for _, m := range members[:limit] {
				qualifiedIDs = append(qualifiedIDs, m.ParticipantID)
			}
		}

		if len(qualifiedIDs) < 2 {
			t.Status = "Finished"
		} else {
			if err := db.DB.Transaction(func(tx *gorm.DB) error {
				if err := generateBracketMatches(tx, t.ID, qualifiedIDs, 1); err != nil {
					return err
				}
				// Award Qualification Points (8)
				var participants []models.TournamentParticipant
				if err := tx.Where("tournament_id = ? AND participant_id IN ?", t.ID, qualifiedIDs).Find(&participants).Error; err != nil {
					return err
				}
				for i := range participants {
					participants[i].AwardQualification()
					if err := tx.Save(&participants[i]).Error; err != nil {
						return err
					}
				}
				t.Status = "BracketInProgress"
				return tx.Save(&t).Error
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	case "BracketInProgress":
		// Check if the final match is completed
		maxRound := 0
		for _, m := range t.Matches {
			if m.Phase == "Bracket" && m.Round > maxRound {
				maxRound = m.Round
			}
		}

		allFinalsFinished := true
		for _, m := range t.Matches {
			if m.Phase == "Bracket" && m.Round == maxRound {
				if m.WinnerID == nil {
					allFinalsFinished = false
					break
				}
			}
		}

		if allFinalsFinished {
			t.Status = "Finished"
			// Award Podium Points
			finalMatch := models.Match{}
			for _, m := range t.Matches {
				if m.Phase == "Bracket" && m.Round == maxRound {
					finalMatch = m
					break
				}
			}

			if finalMatch.WinnerID != nil {
				// 1st Place
				var winnerTP models.TournamentParticipant
				if err := db.DB.Where("tournament_id = ? AND participant_id = ?", t.ID, *finalMatch.WinnerID).First(&winnerTP).Error; err == nil {
					winnerTP.AwardPodium(1)
					db.DB.Save(&winnerTP)
				}

				// 2nd Place
				runnerUpID := finalMatch.Player1ID
				if *finalMatch.WinnerID == *finalMatch.Player1ID {
					runnerUpID = finalMatch.Player2ID
				}
				if runnerUpID != nil {
					var runnerUpTP models.TournamentParticipant
					if err := db.DB.Where("tournament_id = ? AND participant_id = ?", t.ID, *runnerUpID).First(&runnerUpTP).Error; err == nil {
						runnerUpTP.AwardPodium(2)
						db.DB.Save(&runnerUpTP)
					}
				}

				// 3rd Place (Best SF loser)
				if maxRound > 1 {
					var sfLosers []uint
					for _, m := range t.Matches {
						if m.Phase == "Bracket" && m.Round == maxRound-1 {
							loserID := m.Player1ID
							if m.WinnerID != nil && *m.WinnerID == *m.Player1ID {
								loserID = m.Player2ID
							}
							if loserID != nil {
								sfLosers = append(sfLosers, *loserID)
							}
						}
					}
					if len(sfLosers) > 0 {
						var bestTP models.TournamentParticipant
						if err := db.DB.Where("tournament_id = ? AND participant_id IN ?", t.ID, sfLosers).
							Order("wins DESC, points DESC").First(&bestTP).Error; err == nil {
							bestTP.AwardPodium(3)
							db.DB.Save(&bestTP)
						}
					}
				}
			}

			db.DB.Save(&t)
		} else {
			http.Error(w, "Final match is not finished yet", http.StatusBadRequest)
			return
		}

	default:
		t.Status = "Finished"
		db.DB.Save(&t)
	}

	// Refetch with all preloads for the response
	db.DB.Preload("Matches.Player1").Preload("Matches.Player2").Preload("TournamentParticipants.Participant").First(&t, tourID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

// ResetTournament clears matches and resets participant stats and groups
func ResetTournament(w http.ResponseWriter, r *http.Request) {
	tourIDStr := chi.URLParam(r, "id")
	tourID, _ := strconv.Atoi(tourIDStr)

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var t models.Tournament
		if err := tx.First(&t, tourID).Error; err != nil {
			return err
		}

		// 1. Delete all matches for this tournament
		if err := tx.Where("tournament_id = ?", tourID).Delete(&models.Match{}).Error; err != nil {
			return err
		}

		// 2. Reset all TournamentParticipant stats and group
		if err := tx.Model(&models.TournamentParticipant{}).
			Where("tournament_id = ?", tourID).
			Updates(map[string]interface{}{
				"group":           "",
				"wins":            0,
				"losses":          0,
				"draws":           0,
				"points":          0,
				"spin_finishes":   0,
				"burst_finishes":  0,
				"over_finishes":   0,
				"xtreme_finishes": 0,
			}).Error; err != nil {
			return err
		}

		// 3. Reset tournament status
		t.Status = "Created"
		return tx.Save(&t).Error
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "Created"})
}

// ArchiveTournament soft-deletes a tournament
func ArchiveTournament(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.Model(&models.Tournament{}).Where("id = ?", idStr).Update("is_archived", true).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "archived"}`))
}

// UnarchiveTournament unsoft-deletes a tournament
func UnarchiveTournament(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.Model(&models.Tournament{}).Where("id = ?", idStr).Update("is_archived", false).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "restored"}`))
}
