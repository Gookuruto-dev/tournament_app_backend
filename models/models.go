package models

import (
	"time"

	"gorm.io/gorm"
)

// Participant represents a player in the league.
type Participant struct {
	gorm.Model
	Nickname   string `gorm:"uniqueIndex;not null" json:"nickname"`
	Avatar     string `json:"avatar"`
	IsArchived bool   `gorm:"default:false" json:"is_archived"`
}

// Deck represents a player's set of Beyblades for a match.
type Deck struct {
	gorm.Model
	ParticipantID uint       `json:"participant_id"`
	Name          string     `json:"name"`
	Beyblades     []Beyblade `gorm:"foreignKey:DeckID" json:"beyblades"`
}

// Beyblade represents a single combination of parts.
type Beyblade struct {
	gorm.Model
	DeckID  uint   `json:"deck_id"`
	Blade   string `json:"blade"`   // e.g. "Dran Sword"
	Ratchet string `json:"ratchet"` // e.g. "3-60"
	Bit     string `json:"bit"`     // e.g. "Flat"
	// Combined name could be helpful, e.g. "Dran Sword 3-60 Flat"
}

// TournamentParticipant represents the link between a tournament and a participant, including stats and group.
type TournamentParticipant struct {
	gorm.Model
	TournamentID  uint        `json:"tournament_id"`
	ParticipantID uint        `json:"participant_id"`
	Participant   Participant `gorm:"foreignKey:ParticipantID" json:"participant"`
	Group         string      `json:"group"` // "A", "B", etc.
	Wins          int         `json:"wins"`
	Losses        int         `json:"losses"`
	Draws         int         `json:"draws"`
	Points        int         `json:"points"` // e.g., 3 for win, 1 for draw
	// Finish Stats
	SpinFinishes   int `json:"spin_finishes"`
	BurstFinishes  int `json:"burst_finishes"`
	OverFinishes   int `json:"over_finishes"`
	XtremeFinishes int `json:"xtreme_finishes"`
}

// Tournament represents a single event.
type Tournament struct {
	gorm.Model
	Name                   string                  `json:"name"`
	Date                   time.Time               `json:"date"`
	Status                 string                  `json:"status"` // Created, GroupsGenerated, InProgress, BracketInProgress, Finished
	IsArchived             bool                    `gorm:"default:false" json:"is_archived"`
	Participants           []Participant           `gorm:"many2many:tournament_participants_old;" json:"-"` // Deprecated or kept for compat, prefer TournamentParticipants
	TournamentParticipants []TournamentParticipant `gorm:"foreignKey:TournamentID" json:"tournament_participants"`
	Matches                []Match                 `gorm:"foreignKey:TournamentID" json:"matches"`
}

// Match represents a battle between two players.
type Match struct {
	gorm.Model
	TournamentID uint         `json:"tournament_id"`
	Player1ID    *uint        `json:"player1_id"`
	Player2ID    *uint        `json:"player2_id"`
	Player1      *Participant `gorm:"foreignKey:Player1ID" json:"player1"`
	Player2      *Participant `gorm:"foreignKey:Player2ID" json:"player2"`

	ScoreP1  int   `json:"score_p1"`
	ScoreP2  int   `json:"score_p2"`
	WinnerID *uint `json:"winner_id"` // Nullable if draw/ongoing

	Phase string `json:"phase"` // Group, Bracket
	Round int    `json:"round"` // Round number

	NextMatchID   *uint `json:"next_match_id"`   // Future round match
	NextMatchSlot int   `json:"next_match_slot"` // 1 for Player1, 2 for Player2

	// Could store individual round results (spin/burst/extreme) as a JSON blob or separate table if detailed history is needed.
	// For "start simple", maybe just scores? But tracking logic (Burst=2) might need logs.
	// Let's add a serialized field for MatchLog if needed later.
}
