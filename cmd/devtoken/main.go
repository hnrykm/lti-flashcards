// Command devtoken prints a session token for local curl testing without an
// LTI launch. It needs the same SESSION_SECRET as the API.
//
//	SESSION_SECRET=... go run ./cmd/devtoken -course 1 -instructor
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
	course := flag.Int64("course", 1, "course ID (1 is the seeded dev course)")
	user := flag.String("user", "dev-user", "user ID")
	instructor := flag.Bool("instructor", false, "issue an instructor token")
	flag.Parse()

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
