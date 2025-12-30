package main

import (
	"bbx_tournament/db"
	"bbx_tournament/models"
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize DB (Directly, since we are in a cmd script)
	// Assuming running from root: go run ./cmd/seed/main.go
	var err error
	db.DB, err = gorm.Open(sqlite.Open("tournament.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	fmt.Println("Seeding Mock Data...")

	// 1. Create Participants (32 dummy users)
	var participants []models.Participant
	for i := 1; i <= 32; i++ {
		participants = append(participants, models.Participant{
			Nickname: fmt.Sprintf("Blader_%d", i),
			Avatar:   fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=Blader_%d", i),
		})
	}

	for _, p := range participants {
		var existing models.Participant
		if err := db.DB.Where("nickname = ?", p.Nickname).First(&existing).Error; err == nil {
			// fmt.Printf("Participant %s already exists, skipping.\n", p.Nickname)
		} else {
			db.DB.Create(&p)
			fmt.Printf("Created Participant: %s\n", p.Nickname)
		}
	}

	// Re-fetch all to get IDs
	var allParticipants []models.Participant
	db.DB.Find(&allParticipants)

	if len(allParticipants) < 4 {
		log.Fatal("Not enough participants to seed tournament (need at least 4)")
	}

	// 2. Create a Mock Tournament
	mockTournament := models.Tournament{
		Name:   "Mock Blade Battle " + time.Now().Format("15:04"),
		Date:   time.Now(),
		Status: "Created",
	}

	db.DB.Create(&mockTournament)
	fmt.Printf("Created Tournament: %s\n", mockTournament.Name)

	// 3. Add Participants to Tournament
	for _, p := range allParticipants {
		tp := models.TournamentParticipant{
			TournamentID:  mockTournament.ID,
			ParticipantID: p.ID,
		}
		db.DB.Create(&tp)
	}

	fmt.Println("Added participants to tournament via TournamentParticipant table.")
	fmt.Println("Seeding Complete. Restart backend if needed.")
}
