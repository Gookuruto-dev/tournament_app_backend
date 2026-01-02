package main

import (
	"bbx_tournament/db"
	"bbx_tournament/handlers"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	db.InitDB("tournament.db")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Basic CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // For dev
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/participants", handlers.GetParticipants)
	r.Get("/stats", handlers.GetLeagueStats)
	r.Post("/participants", handlers.CreateParticipant)
	r.Post("/participants/{id}/archive", handlers.ArchiveParticipant)

	r.Get("/tournaments", handlers.GetTournaments)
	r.Post("/tournaments", handlers.CreateTournament)
	r.Post("/tournaments/{id}/archive", handlers.ArchiveTournament)
	r.Post("/tournaments/{id}/unarchive", handlers.UnarchiveTournament)
	r.Get("/tournaments/{id}", handlers.GetTournamentDetails)
	r.Post("/tournaments/{id}/participants", handlers.AddParticipantToTournament)
	r.Delete("/tournaments/{id}/participants/{participantId}", handlers.RemoveParticipantFromTournament)
	r.Post("/tournaments/{id}/start", handlers.StartTournament) // Deprecated but kept
	r.Post("/tournaments/{id}/groups", handlers.GenerateGroups)
	r.Post("/tournaments/{id}/matches", handlers.GenerateMatches)
	r.Post("/tournaments/{id}/advance", handlers.AdvanceTournamentPhase)
	r.Post("/tournaments/{id}/reset", handlers.ResetTournament)
	r.Post("/matches/{id}/score", handlers.UpdateMatchScore)
	r.Post("/matches/{id}/reset", handlers.ResetMatch)
	r.Post("/matches/{id}/manual", handlers.ManualMatchScore)

	fmt.Println("BBX Tournament App Backend Service Started on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
