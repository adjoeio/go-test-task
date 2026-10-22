package feed

import "tripmiles/deals"

func (f *Feed) budgetChecks(ds []deals.Deal) []deals.Deal {
	// TODO: Return only the deals that still have hotel stays left.
	// Expectation:
	//   - Hide a deal when one more booking would go over its DailyLimit or TotalLimit,
	//     or over its partner's DailyLimit. Count every booking, confirmed and unconfirmed.
	//   - Send at most 1 request to the Deals Service per app open.
	//   - If the Deals Service does not answer, do not go over a limit,
	//     but keep deals that clearly have stays left.
	return ds
}
