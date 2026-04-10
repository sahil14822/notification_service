package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"notification-service/handlers"
	store "notification-service/repository"
	"notification-service/services"

	_ "notification-service/docs" // generated swagger docs

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Notification Service API
// @version 1.0
// @description This is a notification service backend.
// @host localhost:8080
// @BasePath /

func main() {
	// Auto-load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; assuming environment variables are set globally")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("FATAL: MONGO_URI environment variable is not set!")
	}

	// Connect to MongoDB and initialize the store.
	ctx := context.Background()
	s, err := store.New(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB (%s): %v", mongoURI, err)
	}
	defer func() {
		if err := s.Close(ctx); err != nil {
			log.Printf("Error closing MongoDB connection: %v", err)
		}
	}()
	fmt.Println("Connected to MongoDB:", mongoURI)

	// Initialize service layer with the store.
	svc := services.New(s)

	// Initialize handlers with the service.
	h := handlers.New(svc)

	// Register routes.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Template routes

	mux.HandleFunc("POST /templates", h.CreateTemplate)
	mux.HandleFunc("GET /templates", h.ListTemplates)
	mux.HandleFunc("GET /templates/{id}", h.GetTemplate)
	mux.HandleFunc("PUT /templates/{id}", h.UpdateTemplate)
	mux.HandleFunc("DELETE /templates/{id}", h.DeleteTemplate)
	// Notification routes
	mux.HandleFunc("POST /notifications", h.CreateNotification)
	mux.HandleFunc("GET /notifications/user/{user_id}", h.GetUserNotifications)
	mux.HandleFunc("PATCH /notifications/{id}/read", h.MarkAsRead)

	// Swagger route
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"), //The url pointing to API definition
	))

	fmt.Println("🚀 Notification Service running on http://localhost:8080")
	// fmt.Println("   🌐 Dashboard:  http://localhost:8080/")
	fmt.Println("   GET  /health")
	fmt.Println("   POST /templates")
	fmt.Println("   GET  /templates")
	fmt.Println("   GET  /templates/{id}")
	fmt.Println("   PUT  /templates/{id}")
	fmt.Println("   DELETE /templates/{id}")
	fmt.Println("   POST /notifications")
	fmt.Println("   GET  /notifications/user/{user_id}")
	fmt.Println("   PATCH  /notifications/{id}/read")
	fmt.Println("   GET    /swagger/index.html")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
