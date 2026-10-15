package middleware

import (
	"net/url"
	"sync"
	"time"
	_ "time/tzdata" // embed the zone database: the Alpine image ships without one

	"github.com/gofiber/fiber/v2"
)

// TimezoneCookieName holds the visitor's IANA zone (e.g. "Europe/Istanbul"),
// set by app.js from the browser's Intl API.
const TimezoneCookieName = "tz"

// DefaultTimezone is used until the browser has reported its zone (the very
// first request) or when the cookie is missing/invalid. Almost every user is
// in Turkey, so that's the sensible guess.
const DefaultTimezone = "Europe/Istanbul"

const timezoneContextKey = "tz"

var (
	defaultLocation = mustLoad(DefaultTimezone)
	locationCache   sync.Map // zone name -> *time.Location
)

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// LoadTimezone resolves the visitor's timezone from the tz cookie and stores
// it for handlers to read via Location, so "today" means the visitor's
// today, not the server's (UTC).
func LoadTimezone() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(timezoneContextKey, ResolveLocation(c.Cookies(TimezoneCookieName)))
		return c.Next()
	}
}

// ResolveLocation turns a (URL-encoded) zone name into a location, falling
// back to DefaultTimezone for anything empty, unknown or suspicious.
func ResolveLocation(raw string) *time.Location {
	name, err := url.PathUnescape(raw)
	if err != nil || name == "" || len(name) > 64 {
		return defaultLocation
	}
	if loc, ok := locationCache.Load(name); ok {
		return loc.(*time.Location)
	}
	loc, err := time.LoadLocation(name)
	if err != nil || name == "Local" {
		return defaultLocation
	}
	locationCache.Store(name, loc)
	return loc
}

// Location returns the visitor's timezone resolved by LoadTimezone.
func Location(c *fiber.Ctx) *time.Location {
	if loc, ok := c.Locals(timezoneContextKey).(*time.Location); ok {
		return loc
	}
	return defaultLocation
}
