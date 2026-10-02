// Package promise stores what we promised a user when we showed them an offer.
// Owned by Offers Core (your team).
package promise

import "offerhub/platform"

// Promise is written once, the first time a user is shown a campaign.
// Coins per milestone and the CPI are captured at that moment; later
// campaign changes by the advertiser do not change an existing promise.
type Promise struct {
	UserID     string
	CampaignID string
	CPIMicros  int64
	Coins      map[string]int64 // milestone name -> coins
}

type Store struct{ kv *platform.KV }

func NewStore(kv *platform.KV) *Store { return &Store{kv: kv} }

func key(userID, campaignID string) string { return "promise|" + userID + "|" + campaignID }

// Commit saves p unless a promise for this user and campaign already exists.
// It returns the promise that is in force (the existing one, if any).
func (s *Store) Commit(p Promise) Promise {
	if s.kv.PutIf(key(p.UserID, p.CampaignID), p, 0) {
		return p
	}
	existing, _ := s.Get(p.UserID, p.CampaignID)
	return existing
}

func (s *Store) Get(userID, campaignID string) (Promise, bool) {
	v, _, ok := s.kv.Get(key(userID, campaignID))
	if !ok {
		return Promise{}, false
	}
	return v.(Promise), true
}
