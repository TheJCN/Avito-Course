package main

import (
	"log"
	"net/http"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/handler"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/moderation"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/repository"
	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/service"
)

func main() {
	repo := repository.NewMemoryRepository()
	moderator := moderation.NewClient("http://moderation.internal/check")
	svc := service.NewListingService(repo, moderator)
	h := handler.NewHandler(svc)

	mux := http.NewServeMux()
	h.Register(mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
