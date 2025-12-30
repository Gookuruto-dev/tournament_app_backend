package handlers

import (
	"bbx_tournament/models"

	"gorm.io/gorm"
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
				p1ID := players[i]
				p2ID := players[n-1-i]

				// Skip matches with dummy player (id 0)
				if p1ID != 0 && p2ID != 0 {
					p1 := p1ID
					p2 := p2ID
					matches = append(matches, models.Match{
						TournamentID: tournamentID,
						Player1ID:    &p1,
						Player2ID:    &p2,
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

// generateBracketMatches creates the entire single-elimination bracket tree and saves it to the DB
func generateBracketMatches(tx *gorm.DB, tournamentID uint, playerIDs []uint, startRound int) error {
	n := len(playerIDs)
	if n < 2 {
		return nil
	}

	// 1. Calculate the number of rounds
	// We assume n players qualify. Rounds = ceil(log2(n))
	numRounds := 0
	for (1 << numRounds) < n {
		numRounds++
	}

	// 2. Generate matches round by round in REVERSE (Final first)
	// Track matches of the "next" round to link them as parents
	var nextRoundMatches []models.Match

	for r := numRounds; r >= 1; r-- {
		numMatchesInRound := 1 << (numRounds - r)
		var currentRoundMatches []models.Match

		for i := 0; i < numMatchesInRound; i++ {
			m := models.Match{
				TournamentID: tournamentID,
				Phase:        "Bracket",
				Round:        r + startRound - 1,
			}

			// Link to parent match (from logically subsequent round)
			if len(nextRoundMatches) > 0 {
				parentMatch := nextRoundMatches[i/2]
				m.NextMatchID = &parentMatch.ID
				m.NextMatchSlot = (i % 2) + 1
			}

			// If it's the first round (relative to bracket start), assign players
			if r == 1 {
				p1Idx := i * 2
				p2Idx := i*2 + 1

				if p1Idx < n {
					p1 := playerIDs[p1Idx]
					m.Player1ID = &p1
				}
				if p2Idx < n {
					p2 := playerIDs[p2Idx]
					m.Player2ID = &p2
				}
			}

			// Save to get ID
			if err := tx.Create(&m).Error; err != nil {
				return err
			}
			currentRoundMatches = append(currentRoundMatches, m)
		}
		nextRoundMatches = currentRoundMatches
	}

	return nil
}
