package db

import (
	"bbx_tournament/models"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error

	dsn := os.Getenv("DATABASE_URL")

	if dsn != "" {
		// ✅ PostgreSQL (Render / Production)
		log.Println("Using PostgreSQL database")
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else {
		// 🧪 SQLite fallback (Local / Dev)
		log.Println("DATABASE_URL not found, using SQLite")
		DB, err = gorm.Open(sqlite.Open("tournament.db"), &gorm.Config{})
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate
	err = DB.AutoMigrate(
		&models.Participant{},
		&models.Deck{},
		&models.Beyblade{},
		&models.Tournament{},
		&models.TournamentParticipant{},
		&models.Match{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database initialized successfully.")
}
