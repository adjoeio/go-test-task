package feed

import "tripmiles/deals"

// dealBudgetCheck removes each deal that has used its daily budget or its total budget.
//
// For each deal, it sends four requests to the shared counter service:
// confirmed spend today, bookings today, confirmed spend in total, and bookings in total.
func (f *Feed) dealBudgetCheck(ds []deals.Deal) []deals.Deal {
	var keep []deals.Deal
	for _, d := range ds {
		confirmedToday, err1 := f.Stats.ConfirmedTodayCents(d.ID)
		bookingsToday, err2 := f.Stats.BookingsToday(d.ID)
		confirmedTotal, err3 := f.Stats.ConfirmedTotalCents(d.ID)
		bookingsTotal, err4 := f.Stats.BookingsTotal(d.ID)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			// The service did not answer. The check keeps the deal.
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

// budgetUsedUp returns true if a budget is used.
//
// First, it compares the confirmed spend with the budget.
// Confirmations arrive late, so it also makes a backup estimate:
// the number of bookings multiplied by the price.
// The estimate counts only if it is at least 1.5 times the budget.
// The 1.5 factor stops the check from removing deals too early.
func budgetUsedUp(budgetCents, confirmedCents, bookings, priceCents int64) bool {
	if budgetCents == 0 {
		return false // There is no budget.
	}
	if confirmedCents >= budgetCents {
		return true
	}
	estimateCents := bookings * priceCents
	return estimateCents*2 >= budgetCents*3 // estimate >= 1.5 x budget
}

// partnerLimitCheck removes all deals of each partner that has used its partner daily limit.
//
// For each deal, it sends one request to the shared counter service.
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
