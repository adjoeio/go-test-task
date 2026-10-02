// Package stats is the shared counter service. Owned by the Platform team.
//
// It remembers how many bookings each deal has had and how much money partners
// have confirmed. The app runs on several servers, and all of them use this one
// service. Each server talks to it through its own Client.
//
// Every question a Client asks travels over the network. In real life each
// question takes about 1 ms, and sometimes there is no answer. The simulator
// counts the questions and adds 1 ms for each. Recording a booking is not a question.
package stats

import (
	"errors"
	"sync"
)

// ErrNoAnswer means the service did not answer this question.
var ErrNoAnswer = errors.New("the shared counter service did not answer")

type dealCounters struct {
	bookingsToday, bookingsTotal       int64
	bookedTodayCents, bookedTotalCents int64 // value of every booking, confirmed or not
	confirmedTodayCents                int64 // value the partner has confirmed so far
	confirmedTotalCents                int64
}

type booking struct {
	dealID, partnerID string
	cents             int64
}

// Service is the shared counter service itself.
type Service struct {
	// FailEveryNth makes every Nth question of each Client get no answer. 0 means always answer.
	FailEveryNth int

	mu                sync.Mutex
	deals             map[string]*dealCounters
	partnerBooked     map[string]int64
	partnerConfirmed  map[string]int64
	waitingForConfirm []booking
}

func NewService() *Service {
	return &Service{
		deals:            map[string]*dealCounters{},
		partnerBooked:    map[string]int64{},
		partnerConfirmed: map[string]int64{},
	}
}

func (s *Service) deal(id string) *dealCounters {
	if s.deals[id] == nil {
		s.deals[id] = &dealCounters{}
	}
	return s.deals[id]
}

// ConfirmAllButNewest is called by partners. They confirm every waiting booking
// except the newest n, because confirmations arrive a few minutes late.
func (s *Service) ConfirmAllButNewest(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.waitingForConfirm) > n {
		b := s.waitingForConfirm[0]
		s.waitingForConfirm = s.waitingForConfirm[1:]
		d := s.deal(b.dealID)
		d.confirmedTodayCents += b.cents
		d.confirmedTotalCents += b.cents
		s.partnerConfirmed[b.partnerID] += b.cents
	}
}

// Client is one app server's connection to the shared service.
type Client struct {
	svc   *Service
	calls int
}

func (s *Service) NewClient() *Client { return &Client{svc: s} }

// Calls is how many questions this Client has asked so far.
func (c *Client) Calls() int { return c.calls }

func (c *Client) ask() error {
	c.calls++
	if c.svc.FailEveryNth > 0 && c.calls%c.svc.FailEveryNth == 0 {
		return ErrNoAnswer
	}
	return nil
}

// RecordBooking adds one booking and the price the partner pays for it.
// The booking waits for the partner's confirmation.
func (c *Client) RecordBooking(dealID, partnerID string, cents int64) {
	s := c.svc
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.deal(dealID)
	d.bookingsToday++
	d.bookingsTotal++
	d.bookedTodayCents += cents
	d.bookedTotalCents += cents
	s.partnerBooked[partnerID] += cents
	s.waitingForConfirm = append(s.waitingForConfirm, booking{dealID, partnerID, cents})
}

// The next five methods each ask one question about one deal or partner.

func (c *Client) ConfirmedTodayCents(dealID string) (int64, error) {
	return c.one(func(s *Service) int64 { return s.deal(dealID).confirmedTodayCents })
}

func (c *Client) BookingsToday(dealID string) (int64, error) {
	return c.one(func(s *Service) int64 { return s.deal(dealID).bookingsToday })
}

func (c *Client) ConfirmedTotalCents(dealID string) (int64, error) {
	return c.one(func(s *Service) int64 { return s.deal(dealID).confirmedTotalCents })
}

func (c *Client) BookingsTotal(dealID string) (int64, error) {
	return c.one(func(s *Service) int64 { return s.deal(dealID).bookingsTotal })
}

func (c *Client) PartnerConfirmedTodayCents(partnerID string) (int64, error) {
	return c.one(func(s *Service) int64 { return s.partnerConfirmed[partnerID] })
}

func (c *Client) one(read func(*Service) int64) (int64, error) {
	if err := c.ask(); err != nil {
		return 0, err
	}
	c.svc.mu.Lock()
	defer c.svc.mu.Unlock()
	return read(c.svc), nil
}

// DealNumbers are all the numbers the service has for one deal.
type DealNumbers struct {
	BookingsToday, BookingsTotal       int64
	BookedTodayCents, BookedTotalCents int64 // value of every booking, confirmed or not
	ConfirmedTodayCents                int64
	ConfirmedTotalCents                int64
}

// Snapshot is the answer to one big question about many deals and partners at once.
type Snapshot struct {
	Deals                      map[string]DealNumbers
	PartnerBookedTodayCents    map[string]int64 // value of every booking, confirmed or not
	PartnerConfirmedTodayCents map[string]int64
}

// Snapshot asks one question and gets the numbers for all these deals and partners.
func (c *Client) Snapshot(dealIDs, partnerIDs []string) (Snapshot, error) {
	if err := c.ask(); err != nil {
		return Snapshot{}, err
	}
	s := c.svc
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := Snapshot{
		Deals:                      map[string]DealNumbers{},
		PartnerBookedTodayCents:    map[string]int64{},
		PartnerConfirmedTodayCents: map[string]int64{},
	}
	for _, id := range dealIDs {
		d := s.deal(id)
		snap.Deals[id] = DealNumbers{
			BookingsToday: d.bookingsToday, BookingsTotal: d.bookingsTotal,
			BookedTodayCents: d.bookedTodayCents, BookedTotalCents: d.bookedTotalCents,
			ConfirmedTodayCents: d.confirmedTodayCents, ConfirmedTotalCents: d.confirmedTotalCents,
		}
	}
	for _, id := range partnerIDs {
		snap.PartnerBookedTodayCents[id] = s.partnerBooked[id]
		snap.PartnerConfirmedTodayCents[id] = s.partnerConfirmed[id]
	}
	return snap, nil
}

// TrueNumbers is used only by the simulator to check the results of the day.
// It reads the real numbers directly. App code must ask through a Client.
func (s *Service) TrueNumbers(dealID string) DealNumbers {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.deal(dealID)
	return DealNumbers{
		BookingsToday: d.bookingsToday, BookingsTotal: d.bookingsTotal,
		BookedTodayCents: d.bookedTodayCents, BookedTotalCents: d.bookedTotalCents,
		ConfirmedTodayCents: d.confirmedTodayCents, ConfirmedTotalCents: d.confirmedTotalCents,
	}
}

// TruePartnerBookedTodayCents is used only by the simulator, like TrueNumbers.
func (s *Service) TruePartnerBookedTodayCents(partnerID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.partnerBooked[partnerID]
}
