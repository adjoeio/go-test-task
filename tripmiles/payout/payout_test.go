package payout

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"tripmiles/lock"
	"tripmiles/platform"
)

func setup() (*platform.KV, *lock.Store) {
	kv := platform.NewKV()
	ls := lock.NewStore(kv)
	ls.Save(lock.Lock{UserID: "u1", DealID: "d1", Miles: map[string]int64{"trip-completed": 300}})
	return kv, ls
}

func TestPaysTheLockedMiles(t *testing.T) {
	kv, ls := setup()
	s := &Service{Locks: ls, KV: kv}
	got, err := s.Pay(Update{ID: "e1", UserID: "u1", DealID: "d1", Goal: "trip-completed"})
	if err != nil || got != 300 || s.Balance("u1") != 300 {
		t.Fatalf("paid %d (err %v), balance %d; want 300", got, err, s.Balance("u1"))
	}
}

func TestNoBookingNoMiles(t *testing.T) {
	kv, ls := setup()
	s := &Service{Locks: ls, KV: kv}
	if _, err := s.Pay(Update{ID: "e1", UserID: "u2", DealID: "d1", Goal: "trip-completed"}); err == nil {
		t.Fatal("paid without a booking")
	}
	if s.Balance("u2") != 0 {
		t.Fatal("balance changed without a booking")
	}
}

func TestRedeliveredUpdateIsPaidOnce(t *testing.T) {
	kv, ls := setup()
	s := &Service{Locks: ls, KV: kv}
	u := Update{ID: "e1", UserID: "u1", DealID: "d1", Goal: "trip-completed"}
	s.Pay(u)
	s.Pay(u)
	if s.Balance("u1") != 300 {
		t.Fatalf("balance %d, want 300", s.Balance("u1"))
	}
}

// Four server instances receive the same updates at the same time.
func TestConcurrentInstancesPayOnce(t *testing.T) {
	kv, ls := setup()
	kv.Latency = time.Millisecond
	goals := map[string]int64{}
	for i := 0; i < 20; i++ {
		goals[fmt.Sprintf("g%d", i)] = 10
	}
	ls.Save(lock.Lock{UserID: "u9", DealID: "d9", Miles: goals})

	var wg sync.WaitGroup
	for inst := 0; inst < 4; inst++ {
		s := &Service{Locks: ls, KV: kv}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for g := range goals {
				s.Pay(Update{ID: g, UserID: "u9", DealID: "d9", Goal: g})
			}
		}()
	}
	wg.Wait()
	b := (&Service{Locks: ls, KV: kv}).Balance("u9")
	if b != 200 {
		t.Fatalf("balance %d, want 200 (20 goals x 10 miles, each paid once)", b)
	}
}
