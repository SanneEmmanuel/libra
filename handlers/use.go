package handlers

import (
    "html/template"
    "net/http"
)

func UseHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("templates/use.html")
    if err != nil {
        http.Error(w, "Template error", http.StatusInternalServerError)
        return
    }

    tmpl.Execute(w, nil)
}
