package auth

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

type ctxKey struct{}

// Middleware requires "Authorization: Bearer <session token>" and stores the
// session in c.Locals for handlers to read with FromCtx.
func (s *Signer) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if !ok || token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}
		sess, err := s.Verify(token, time.Now())
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		c.Locals(ctxKey{}, sess)
		return c.Next()
	}
}

// RequireInstructor rejects students with 403. Use it after Middleware.
func RequireInstructor(c fiber.Ctx) error {
	if !FromCtx(c).Instructor {
		return fiber.NewError(fiber.StatusForbidden, "instructors only")
	}
	return c.Next()
}

// FromCtx returns the session that Middleware stored. It is the zero Session
// if Middleware didn't run, which matches no course.
func FromCtx(c fiber.Ctx) Session {
	sess, _ := c.Locals(ctxKey{}).(Session)
	return sess
}
