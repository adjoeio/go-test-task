// Package stats is the shared counter service. The Platform team owns it.
//
// The service counts the bookings of each deal. It also keeps the confirmed
// spend: the money that partners have confirmed. The app runs on several
// servers. All servers use this one service. Each server uses its own Client.
//
// Each request from a Client goes over the network. In production, one request
// takes about 1 ms. Sometimes the service does not answer a request.
// The simulator counts the requests and adds 1 ms for each request.
// Recording a booking is not counted as a request.
package stats

import (
	"errors"
	"sync"
)

// ErrNoAnswer means that the service did not answer the request.
var ErrNoAnswer = errors.New("the shared counter service did not answer")

type dealCounters struct {
	bookingsToday, bookingsTotal       int64
	bookedTodayCents, bookedTotalCents int64 // value of all bookings, confirmed and unconfirmed
	confirmedTodayCents                int64 // confirmed spend
	confirmedTotalCents                int64
}

type booking struct {
	dealID, partnerID string
	cents             int64
}

// Service is the shared counter service.
type Service struct {
	// If FailEveryNth is N, the service does not answer every Nth request of each Client.
	// If FailEveryNth is 0, the service answers every request.
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

// ConfirmAllButNewest confirms all unconfirmed bookings except the newest n.
// Partners call it. The newest bookings stay unconfirmed because
// confirmations arrive a few minutes after the booking.
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

// Client is the connection of one app server to the shared counter service.
type Client struct {
	svc   *Service
	calls int
}

func (s *Service) NewClient() *Client { return &Client{svc: s} }

// Calls returns the number of requests that this Client has sent.
func (c *Client) Calls() int { return c.calls }

func (c *Client) ask() error {
	c.calls++
	if c.svc.FailEveryNth > 0 && c.calls%c.svc.FailEveryNth == 0 {
		return ErrNoAnswer
	}
	return nil
}

// RecordBooking adds one booking and the amount that the partner pays for it.
// The booking stays unconfirmed until the partner confirms it.
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

// Each of the next five methods sends one request about one deal or one partner.
// Each method returns confirmed spend or a booking count. It does not return
// the value of unconfirmed bookings.

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

// DealNumbers holds all the numbers that the service has for one deal.
type DealNumbers struct {
	BookingsToday, BookingsTotal       int64
	BookedTodayCents, BookedTotalCents int64 // value of all bookings, confirmed and unconfirmed
	ConfirmedTodayCents                int64 // confirmed spend today
	ConfirmedTotalCents                int64 // confirmed spend in total
}

// Snapshot holds the numbers for many deals and partners. One request returns it.
type Snapshot struct {
	Deals                      map[string]DealNumbers
	PartnerBookedTodayCents    map[string]int64 // value of all bookings today, confirmed and unconfirmed
	PartnerConfirmedTodayCents map[string]int64 // confirmed spend today
}

// Snapshot sends one request. It returns the numbers for all these deals and partners.
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

// TrueNumbers is for the simulator only. The simulator uses it to check the day.
// It reads the numbers directly and sends no request. App code must use a Client.
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

// TruePartnerBookedTodayCents is for the simulator only, like TrueNumbers.
func (s *Service) TruePartnerBookedTodayCents(partnerID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.partnerBooked[partnerID]
}
