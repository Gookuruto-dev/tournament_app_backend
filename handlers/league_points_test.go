package handlers

import (
	"bbx_tournament/models"
	"testing"
)

func TestLeaguePointsModelLogic(t *testing.T) {
	tp := models.TournamentParticipant{
		LeaguePoints: 0,
	}

	// 1. Test Participation
	tp.AwardParticipation()
	if tp.LeaguePoints != 5 {
		t.Errorf("AwardParticipation failed: expected 5, got %d", tp.LeaguePoints)
	}

	// 2. Test Qualification
	tp.AwardQualification()
	if tp.LeaguePoints != 13 { // 5 + 8
		t.Errorf("AwardQualification failed: expected 13, got %d", tp.LeaguePoints)
	}

	// 3. Test Bracket Win
	tp.AwardBracketWin()
	if tp.LeaguePoints != 16 { // 13 + 3
		t.Errorf("AwardBracketWin failed: expected 16, got %d", tp.LeaguePoints)
	}

	// 4. Test Podium - 1st
	tp.AwardPodium(1)
	if tp.LeaguePoints != 51 { // 16 + 35
		t.Errorf("AwardPodium(1) failed: expected 51, got %d", tp.LeaguePoints)
	}

	// Reset for 2nd and 3rd place tests
	tp.LeaguePoints = 0

	// 5. Test Podium - 2nd
	tp.AwardPodium(2)
	if tp.LeaguePoints != 19 {
		t.Errorf("AwardPodium(2) failed: expected 19, got %d", tp.LeaguePoints)
	}

	// 6. Test Podium - 3rd
	tp.LeaguePoints = 0
	tp.AwardPodium(3)
	if tp.LeaguePoints != 12 {
		t.Errorf("AwardPodium(3) failed: expected 12, got %d", tp.LeaguePoints)
	}
}
