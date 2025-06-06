package main

import (
    "log"
    "net/http"

    "libra/handlers"
    "github.com/gorilla/mux"
)

func main() {
    router := mux.NewRouter()
    router.HandleFunc("/chatx", handlers.ChatXHandler).Methods("POST")
    router.HandleFunc("/use", handlers.UseHandler).Methods("GET")

    log.Println("Server running on port 10000")
    log.Fatal(http.ListenAndServe(":10000", router))
}
