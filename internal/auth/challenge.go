package auth

import (
	"sync"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go"
)

const challengeTTL = 5 * time.Minute

// ChallengeService generates and verifies Altcha PoW challenges.
type ChallengeService struct {
	hmacKey string
	// Anti-replay: track used challenge signatures
	usedMu   sync.Mutex
	used     map[string]time.Time
	stopUsed chan struct{}
}

func NewChallengeService(hmacKey string) *ChallengeService {
	cs := &ChallengeService{
		hmacKey:  hmacKey,
		used:     make(map[string]time.Time),
		stopUsed: make(chan struct{}),
	}
	go cs.cleanupUsed()
	return cs
}

func (s *ChallengeService) Generate() (altcha.Challenge, error) {
	return altcha.CreateChallenge(altcha.ChallengeOptions{
		HMACKey:   s.hmacKey,
		Algorithm: altcha.SHA256,
		MaxNumber: 100000,
		Expires:   timePtr(time.Now().Add(challengeTTL)),
	})
}

// Verify checks the PoW solution and ensures it hasn't been used before (anti-replay).
func (s *ChallengeService) Verify(payload string) (bool, error) {
	ok, err := altcha.VerifySolution(payload, s.hmacKey, true)
	if err != nil || !ok {
		return false, err
	}

	// Anti-replay: reject if this payload was already used
	s.usedMu.Lock()
	defer s.usedMu.Unlock()

	if _, exists := s.used[payload]; exists {
		return false, nil
	}
	s.used[payload] = time.Now()
	return true, nil
}

func (s *ChallengeService) Stop() {
	close(s.stopUsed)
}

func (s *ChallengeService) cleanupUsed() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopUsed:
			return
		case <-ticker.C:
			s.usedMu.Lock()
			now := time.Now()
			for k, t := range s.used {
				if now.Sub(t) > challengeTTL+time.Minute {
					delete(s.used, k)
				}
			}
			s.usedMu.Unlock()
		}
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
