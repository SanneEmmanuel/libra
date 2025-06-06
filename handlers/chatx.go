package handlers

import (
    "fmt"
    "io"
    "net/http"
    "os"

    "libra/services"
)

func ChatXHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseMultipartForm(10 << 20) // 10MB
    if err != nil {
        http.Error(w, "Unable to parse form", http.StatusBadRequest)
        return
    }

    command := r.FormValue("command")
    file, handler, err := r.FormFile("attachment")
    if err != nil {
        http.Error(w, "Missing attachment", http.StatusBadRequest)
        return
    }
    defer file.Close()

    tempFile, err := os.CreateTemp("", handler.Filename)
    if err != nil {
        http.Error(w, "Unable to save file", http.StatusInternalServerError)
        return
    }
    defer os.Remove(tempFile.Name())

    _, err = io.Copy(tempFile, file)
    if err != nil {
        http.Error(w, "Error saving file", http.StatusInternalServerError)
        return
    }

    tempFile.Seek(0, 0)
    result, err := services.CallDeepSeekAPI(command, tempFile)
    if err != nil {
        http.Error(w, fmt.Sprintf("DeepSeek error: %v", err), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(result)
}
