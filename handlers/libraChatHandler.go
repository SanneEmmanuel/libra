package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "libra/services"
)

func LibraChatHandler(w http.ResponseWriter, r *http.Request) {
    userText := r.URL.Query().Get("q")
    if userText == "" {
        http.Error(w, "Missing query parameter: q", http.StatusBadRequest)
        return
    }

    stream := false
    streamParam := r.URL.Query().Get("stream")
    if streamParam != "" {
        parsedStream, err := strconv.ParseBool(streamParam)
        if err == nil {
            stream = parsedStream
        }
    }

    reply, err := services.LibraChat(userText, stream)
    if err != nil {
        http.Error(w, "Chat error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{
        "response": reply,
    })
}
