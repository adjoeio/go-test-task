package reward

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"offerhub/platform"
	"offerhub/promise"
)

func setup() (*platform.KV, *promise.Store) {
	kv := platform.NewKV()
	ps := promise.NewStore(kv)
	ps.Commit(promise.Promise{UserID: "u1", CampaignID: "c1", Coins: map[string]int64{"level-10": 300}})
	return kv, ps
}

func TestPaysThePromisedCoins(t *testing.T) {
	kv, ps := setup()
	s := &Service{Promises: ps, KV: kv}
	got, err := s.Pay(Event{ID: "e1", UserID: "u1", CampaignID: "c1", Milestone: "level-10"})
	if err != nil || got != 300 || s.Balance("u1") != 300 {
		t.Fatalf("paid %d (err %v), balance %d; want 300", got, err, s.Balance("u1"))
	}
}

func TestNoPromiseNoPayment(t *testing.T) {
	kv, ps := setup()
	s := &Service{Promises: ps, KV: kv}
	if _, err := s.Pay(Event{ID: "e1", UserID: "u2", CampaignID: "c1", Milestone: "level-10"}); err == nil {
		t.Fatal("paid without a promise")
	}
	if s.Balance("u2") != 0 {
		t.Fatal("balance changed without a promise")
	}
}

func TestRedeliveredEventIsPaidOnce(t *testing.T) {
	kv, ps := setup()
	s := &Service{Promises: ps, KV: kv}
	e := Event{ID: "e1", UserID: "u1", CampaignID: "c1", Milestone: "level-10"}
	s.Pay(e)
	s.Pay(e)
	if s.Balance("u1") != 300 {
		t.Fatalf("balance %d, want 300", s.Balance("u1"))
	}
}

// Four server instances receive the same events at the same time.
func TestConcurrentInstancesPayOnce(t *testing.T) {
	kv, ps := setup()
	kv.Latency = time.Millisecond
	milestones := map[string]int64{}
	for i := 0; i < 20; i++ {
		milestones[fmt.Sprintf("m%d", i)] = 10
	}
	ps.Commit(promise.Promise{UserID: "u9", CampaignID: "c9", Coins: milestones})

	var wg sync.WaitGroup
	for inst := 0; inst < 4; inst++ {
		s := &Service{Promises: ps, KV: kv}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for m := range milestones {
				s.Pay(Event{ID: m, UserID: "u9", CampaignID: "c9", Milestone: m})
			}
		}()
	}
	wg.Wait()
	b := (&Service{Promises: ps, KV: kv}).Balance("u9")
	if b != 200 {
		t.Fatalf("balance %d, want 200 (20 milestones x 10 coins, each paid once)", b)
	}
}
