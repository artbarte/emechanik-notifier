package main

import (
	"log"
	"os"
	"time"

	"github.com/artbarte/emechanik-notifier/notifier"
)

func main() {
	n := notifier.Create()
	err := n.Login(os.Getenv("LOGIN"), os.Getenv("PASS"))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Succesfully logged in")
	log.Println("Checking for new posts...")
	n.NotifyAboutLatestPosts(os.Getenv("GURL"))

	// Set up reapeadly chcking if there are new posts
	ticker := time.NewTicker(30 * time.Minute)
	for {
		<-ticker.C
		log.Println("Checking for new posts...")
		n.NotifyAboutLatestPosts(os.Getenv("GURL"))
	}

}
