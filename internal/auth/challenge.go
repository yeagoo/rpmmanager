package auth

import (
	"time"

	altcha "github.com/altcha-org/altcha-lib-go"
)

// ChallengeService generates and verifies Altcha PoW challenges.
type ChallengeService struct {
	hmacKey string
}

func NewChallengeService(hmacKey string) *ChallengeService {
	return &ChallengeService{hmacKey: hmacKey}
}

func (s *ChallengeService) Generate() (altcha.Challenge, error) {
	return altcha.CreateChallenge(altcha.ChallengeOptions{
		HMACKey:   s.hmacKey,
		Algorithm: altcha.SHA256,
		MaxNumber: 100000,
		Expires:   timePtr(time.Now().Add(5 * time.Minute)),
	})
}

func (s *ChallengeService) Verify(payload string) (bool, error) {
	return altcha.VerifySolution(payload, s.hmacKey, true)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
