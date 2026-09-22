package main

import (
	"log"
	"net/http"
	"time"
)

func homePage(w http.ResponseWriter, r *http.Request) {
	// Render the home html page from static folder
	http.ServeFile(w, r, "static/home.html")
}

func coursePage(w http.ResponseWriter, r *http.Request) {
	// Render the course html page
	http.ServeFile(w, r, "static/courses.html")
}

func aboutPage(w http.ResponseWriter, r *http.Request) {
	// Render the about html page
	http.ServeFile(w, r, "static/about.html")
}

func contactPage(w http.ResponseWriter, r *http.Request) {
	// Render the contact html page
	http.ServeFile(w, r, "static/contact.html")
}

func main() {
	http.HandleFunc("/home", homePage)
	http.HandleFunc("/courses", coursePage)
	http.HandleFunc("/about", aboutPage)
	http.HandleFunc("/contact", contactPage)

	server := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           nil,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Fatal(server.ListenAndServe())

}
