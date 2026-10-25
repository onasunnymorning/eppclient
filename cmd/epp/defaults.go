package main

import (
	"crypto/rand"
	"math/big"

	epp "github.com/onasunnymorning/eppclient"
)

const (
	lowerChars  = "abcdefghijklmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars  = "0123456789"
	symbolChars = "!#%*+=?@_" // shell- and XML-friendly
	authLength  = 16
)

func randIndex(n int) int {
	i, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err) // crypto/rand does not fail on supported platforms
	}
	return int(i.Int64())
}

// genAuthInfo returns a random 16-character password with at least one
// lowercase letter, uppercase letter, digit and symbol, which satisfies
// the authInfo rules of most registries.
func genAuthInfo() string {
	classes := []string{lowerChars, upperChars, digitChars, symbolChars}
	all := lowerChars + upperChars + digitChars + symbolChars

	b := make([]byte, authLength)
	for i := range b {
		set := all
		if i < len(classes) {
			set = classes[i]
		}
		b[i] = set[randIndex(len(set))]
	}
	// Shuffle so the guaranteed classes aren't always up front.
	for i := len(b) - 1; i > 0; i-- {
		j := randIndex(i + 1)
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// genContactID returns an ID like "epp-k3f9a2" (within RFC 5733's 3-16 chars).
func genContactID() string {
	const chars = lowerChars + digitChars
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[randIndex(len(chars))]
	}
	return "epp-" + string(b)
}

// testContact is a placeholder contact for test registrations.
type testContact struct {
	ID     string
	Email  string
	Voice  string
	Auth   string
	Postal epp.PostalInfo
}

func newTestContact(email string) testContact {
	if email == "" {
		email = "test@example.com"
	}
	return testContact{
		ID:    genContactID(),
		Email: email,
		Voice: "+1.5555550100",
		Auth:  genAuthInfo(),
		Postal: epp.PostalInfo{
			Name:   "EPP Test",
			Street: "1 Test Street",
			City:   "Testville",
			SP:     "CA",
			PC:     "90001",
			CC:     "US",
		},
	}
}
