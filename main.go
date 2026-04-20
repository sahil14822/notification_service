package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	handlers "notification-service/handlers"
	pb "notification-service/proto"
	store "notification-service/repository"
	"notification-service/services"

	_ "notification-service/docs" // generated swagger docs

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"google.golang.org/grpc"
)

// @title Notification Service API
// @version 1.0
// @description This is a notification service backend backed by Apache Cassandra.
// @host localhost:8080
// @BasePath /

func main() {
	// Auto-load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; assuming environment variables are set globally")
	}

	// ── Cassandra configuration ──────────────────────────────────────────────
	cassandraHosts := os.Getenv("CASSANDRA_HOSTS")
	if cassandraHosts == "" {
		cassandraHosts = "127.0.0.1" // default for local development
		log.Println("CASSANDRA_HOSTS not set; using default: 127.0.0.1")
	}

	cassandraKeyspace := os.Getenv("CASSANDRA_KEYSPACE")
	if cassandraKeyspace == "" {
		cassandraKeyspace = "notification_service" // default keyspace
		log.Println("CASSANDRA_KEYSPACE not set; using default: notification_service")
	}

	// ── Connect to Cassandra and initialise the store ────────────────────────
	ctx := context.Background()
	s, err := store.New(ctx, cassandraHosts, cassandraKeyspace)
	if err != nil {
		log.Fatalf("Failed to connect to Cassandra (%s): %v", cassandraHosts, err)
	}
	defer func() {
		if err := s.Close(ctx); err != nil {
			log.Printf("Error closing Cassandra session: %v", err)
		}
	}()
	fmt.Printf("Connected to Cassandra: hosts=%s  keyspace=%s\n", cassandraHosts, cassandraKeyspace)

	// ── Service and handler layers ───────────────────────────────────────────
	var svc = services.New(s)
	var h = handlers.New(svc)
	// Grpc Server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	grpcHandler := handlers.NewGRPCServer(h)
	pb.RegisterNotificationServiceServer(grpcServer, grpcHandler)

	log.Println("Notification gRPC server running on :50051")

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()
	// ── Routes ───────────────────────────────────────────────────────────────
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

	// Swagger UI
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	fmt.Println("🚀 Notification Service running on http://localhost:8080")
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
