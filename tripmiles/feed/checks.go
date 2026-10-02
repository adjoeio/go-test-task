package feed

import "tripmiles/deals"

// dealBudgetCheck is Check 1. It leaves out deals that have used up their
// daily budget or their total budget.
//
// For each deal it asks the shared counter service four questions:
// confirmed money and bookings, for today and in total.
func (f *Feed) dealBudgetCheck(ds []deals.Deal) []deals.Deal {
	var keep []deals.Deal
	for _, d := range ds {
		confirmedToday, err1 := f.Stats.ConfirmedTodayCents(d.ID)
		bookingsToday, err2 := f.Stats.BookingsToday(d.ID)
		confirmedTotal, err3 := f.Stats.ConfirmedTotalCents(d.ID)
		bookingsTotal, err4 := f.Stats.BookingsTotal(d.ID)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			// The service did not answer. Keep the deal rather than hide it.
			keep = append(keep, d)
			continue
		}
		if budgetUsedUp(d.DailyBudgetCents, confirmedToday, bookingsToday, d.PriceCents) {
			continue
		}
		if budgetUsedUp(d.TotalBudgetCents, confirmedTotal, bookingsTotal, d.PriceCents) {
			continue
		}
		keep = append(keep, d)
	}
	return keep
}

// budgetUsedUp says whether a budget has been reached.
//
// It looks at the money the partner has confirmed. Confirmations arrive late,
// so it also makes a backup guess: bookings times the price. To avoid hiding
// deals too early, the guess only counts above 1.5 times the budget.
func budgetUsedUp(budgetCents, confirmedCents, bookings, priceCents int64) bool {
	if budgetCents == 0 {
		return false // no budget set
	}
	if confirmedCents >= budgetCents {
		return true
	}
	guessCents := bookings * priceCents
	return guessCents*2 >= budgetCents*3 // guess >= 1.5 x budget
}

// partnerLimitCheck is Check 2. It leaves out every deal of a partner that has
// used up its daily limit for all its deals together.
//
// For each deal it asks the shared counter service one question.
func (f *Feed) partnerLimitCheck(ds []deals.Deal) []deals.Deal {
	var keep []deals.Deal
	for _, d := range ds {
		limit := f.Deals.Partner(d.PartnerID).DailyLimitCents
		confirmed, err := f.Stats.PartnerConfirmedTodayCents(d.PartnerID)
		if err != nil || limit == 0 || confirmed < limit {
			keep = append(keep, d)
		}
	}
	return keep
}
