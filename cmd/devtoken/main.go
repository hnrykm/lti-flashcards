// Command devtoken prints a session token for local curl testing without an
// LTI launch. It needs the same SESSION_SECRET as the API.
//
//	SESSION_SECRET=... go run ./cmd/devtoken -course <id> -instructor
//
// The course must already exist; launch the tool from Moodle once to create it.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hnrykm/lti-flashcards/internal/auth"
)

func main() {
	course := flag.Int64("course", 0, "course ID (required; see SELECT id, title FROM courses)")
	user := flag.String("user", "dev-user", "user ID")
	instructor := flag.Bool("instructor", false, "issue an instructor token")
	flag.Parse()
	if *course <= 0 {
		log.Fatal("-course is required: pass a course ID from SELECT id, title FROM courses")
	}

	signer, err := auth.NewSigner([]byte(os.Getenv("SESSION_SECRET")))
	if err != nil {
		log.Fatal(err)
	}
	token, err := signer.Issue(auth.Session{
		CourseID:   *course,
		UserID:     *user,
		Name:       "Dev User",
		Instructor: *instructor,
	}, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token)
}
