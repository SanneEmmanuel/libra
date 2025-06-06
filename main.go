package main

import (
    "log"
    "net/http"

    "github.com/gorilla/mux"
    "libra/handlers"
)

// CORS middleware: allow all origins
func enableCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        // Handle preflight (OPTIONS)
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func main() {
    router := mux.NewRouter()

    // Routes
    router.HandleFunc("/chatx", handlers.ChatXHandler).Methods("POST")
    router.HandleFunc("/use", handlers.UseHandler).Methods("GET")
    router.HandleFunc("/libra", handlers.LibraChatHandler).Methods("GET", "OPTIONS") // New endpoint

    // Apply CORS middleware to all routes
    corsRouter := enableCORS(router)

    log.Println("Server running on port 10000")
    log.Fatal(http.ListenAndServe(":10000", corsRouter))
}
