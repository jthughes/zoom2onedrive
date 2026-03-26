package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	type response struct {
		Error string `json:"error"`
	}

	if err != nil {
		log.Println(err)
	}

	data, err := json.Marshal(response{
		Error: msg,
	})
	w.WriteHeader(code)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}

}
