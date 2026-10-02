// Package reward pays users when they reach a milestone. Owned by Offers Core (your team).
//
// Events come from a queue that delivers at least once: the same event can
// arrive twice, at the same time, on different server instances. Every
// instance shares the same KV store; nothing else is shared between them.
package reward

import (
	"errors"

	"offerhub/catalog"
	"offerhub/platform"
	"offerhub/promise"
)

var (
	ErrNoPromise        = errors.New("no promise for this user and campaign")
	ErrUnknownMilestone = errors.New("milestone not in promise")
)

// Event says a user reached a milestone in a game.
type Event struct {
	ID         string // unique per event; a redelivery has the same ID
	UserID     string
	CampaignID string
	Milestone  string
}

// Service is one server instance. Several run at once against the same KV.
type Service struct {
	Promises    *promise.Store
	Catalog     *catalog.Store // campaign settings as they are right now
	KV          *platform.KV
	MarginPct   int64
	CoinsPerUSD int64
}

func balanceKey(userID string) string { return "balance|" + userID }

// Pay credits the user for e and returns the coins paid by this call.
// Rules:
//   - pay exactly the coins we promised when we showed the offer;
//   - no promise (or milestone not in it) means no payment;
//   - each milestone is paid at most once per user and campaign,
//     however many times or wherever the event is delivered.
func (s *Service) Pay(e Event) (int64, error) {
	// TODO: implement. See the rules above and README.md.
	return 0, errors.New("not implemented")
}

// Balance returns the user's total coins.
func (s *Service) Balance(userID string) int64 {
	v, _, _ := s.KV.Get(balanceKey(userID))
	b, _ := v.(int64)
	return b
}
