package handlers

import (
	"bbx_tournament/models"
)

// generateMatchesFromGroups creates matches based on assigned groups using a Circle Method for rounds
func generateMatchesFromGroups(tournamentID uint, participants []models.TournamentParticipant) []models.Match {
	var matches []models.Match

	// Group participants by their assigned group string
	grouped := make(map[string][]models.TournamentParticipant)
	for _, p := range participants {
		grouped[p.Group] = append(grouped[p.Group], p)
	}

	for groupName, groupMembers := range grouped {
		n := len(groupMembers)
		if n < 2 {
			continue
		}

		// Use a local copy of slice for rotation
		players := make([]uint, n)
		for i, p := range groupMembers {
			players[i] = p.ParticipantID
		}

		// If odd number of players, add a dummy player (0) for Byes
		if n%2 != 0 {
			players = append(players, 0)
			n++
		}

		numRounds := n - 1
		half := n / 2

		for round := 1; round <= numRounds; round++ {
			for i := 0; i < half; i++ {
				p1 := players[i]
				p2 := players[n-1-i]

				// Skip matches with dummy player (id 0)
				if p1 != 0 && p2 != 0 {
					matches = append(matches, models.Match{
						TournamentID: tournamentID,
						Player1ID:    p1,
						Player2ID:    p2,
						Phase:        groupName,
						Round:        round,
						ScoreP1:      0,
						ScoreP2:      0,
					})
				}
			}
			// Rotate players (keep the first one fixed)
			players = append([]uint{players[0]}, append([]uint{players[n-1]}, players[1:n-1]...)...)
		}
	}

	return matches
}

// generateBracketMatches creates a round of a single-elimination bracket
func generateBracketMatches(tournamentID uint, playerIDs []uint, round int) []models.Match {
	var matches []models.Match
	n := len(playerIDs)
	if n < 2 {
		return matches
	}

	// Simple pairing: 0 vs 1, 2 vs 3, etc.
	for i := 0; i < n; i += 2 {
		if i+1 >= n {
			// Odd number of players? Should ideally be handled by Byes earlier,
			// but for now let's just break or handle if needed.
			// Bracket should usually be power of 2 from selection logic.
			break
		}
		p1 := playerIDs[i]
		p2 := playerIDs[i+1]
		matches = append(matches, models.Match{
			TournamentID: tournamentID,
			Player1ID:    p1,
			Player2ID:    p2,
			Phase:        "Bracket",
			Round:        round,
			ScoreP1:      0,
			ScoreP2:      0,
		})
	}

	return matches
}
