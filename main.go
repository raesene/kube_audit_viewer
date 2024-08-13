package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

type LogEntry map[string]interface{}

var (
	logEntries []LogEntry
	logFile    string
	port       int
)

func main() {
	// Parse command line flags
	flag.StringVar(&logFile, "logfile", "", "Path to the Kubernetes audit log file")
	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.Parse()

	if logFile == "" {
		log.Fatal("Please specify a log file using the -logfile flag")
	}

	// Read and parse the log file
	if err := readLogFile(logFile); err != nil {
		log.Fatal(err)
	}

	// Set up HTTP handlers
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/search", searchHandler)

	// Start the web server
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server starting on port %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func readLogFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return err
		}
		logEntries = append(logEntries, entry)
	}

	return scanner.Err()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Kubernetes Audit Log Viewer</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        h1 { color: #333; }
        #search { margin-bottom: 20px; }
        #logs { border: 1px solid #ddd; padding: 10px; }
        pre { white-space: pre-wrap; word-wrap: break-word; }
    </style>
</head>
<body>
    <h1>Kubernetes Audit Log Viewer</h1>
    <div id="search">
        <form action="/search" method="get">
            <input type="text" name="query" placeholder="Search logs...">
            <input type="submit" value="Search">
        </form>
    </div>
    <div id="logs">
        {{range .}}
        <pre>{{.}}</pre>
        {{end}}
    </div>
</body>
</html>
`
	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, logEntries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var results []LogEntry
	for _, entry := range logEntries {
		entryJSON, _ := json.Marshal(entry)
		if strings.Contains(strings.ToLower(string(entryJSON)), strings.ToLower(query)) {
			results = append(results, entry)
		}
	}

	t, err := template.New("index").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>Search Results - Kubernetes Audit Log Viewer</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        h1 { color: #333; }
        #search { margin-bottom: 20px; }
        #logs { border: 1px solid #ddd; padding: 10px; }
        pre { white-space: pre-wrap; word-wrap: break-word; }
    </style>
</head>
<body>
    <h1>Search Results</h1>
    <div id="search">
        <form action="/search" method="get">
            <input type="text" name="query" placeholder="Search logs..." value="{{.Query}}">
            <input type="submit" value="Search">
        </form>
        <a href="/">Back to full log</a>
    </div>
    <div id="logs">
        {{range .Results}}
        <pre>{{.}}</pre>
        {{end}}
    </div>
</body>
</html>
`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Query   string
		Results []LogEntry
	}{
		Query:   query,
		Results: results,
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
