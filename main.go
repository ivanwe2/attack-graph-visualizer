// Command mst-demo serves a browser UI that visualizes how minimum spanning
// trees are built (Kruskal, Prim) plus Dijkstra's shortest-path tree.
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"

	"mst-demo/mst"
)

// The go:embed directive copies the static/ folder into the binary at build
// time, so the program is a single file that works offline.
//
//go:embed static
var staticFiles embed.FS

type runRequest struct {
	Graph     mst.Graph `json:"graph"`
	Algorithm string    `json:"algorithm"`
	Start     string    `json:"start"`
}

func main() {
	// fs.Sub strips the "static/" prefix so index.html is served at "/".
	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	// On some Windows machines the registry maps .js to text/plain, and
	// browsers refuse to run ES modules served with that type.
	if err := mime.AddExtensionType(".js", "text/javascript; charset=utf-8"); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(static))
	mux.HandleFunc("POST /api/run", handleRun) // Go 1.22+ patterns can include the method

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("MST demo: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // refuse bodies over 1 MB

	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "невалиден JSON: " + err.Error()})
		return
	}
	result, err := mst.Run(req.Graph, req.Algorithm, req.Start)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
