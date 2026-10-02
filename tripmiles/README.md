# TripMiles — pre-interview task

TripMiles shows travellers a list of travel deals, such as cheap flights and hotel
nights. Travel companies, which we call **partners**, pay us a fixed price every
time a traveller books one of their deals. Each partner also sets a **daily
budget** for each deal: the most they will pay us for that deal in one day.

You are on the **Offers team**. Your team builds the list of deals that a traveller
sees when they open the app. You must never show a deal whose budget is already
used up, because the partner will not pay for bookings over budget.

The app works today. Under real traffic, though, it asks the shared counter service
far too many questions, it is slow, and some deals go over their budget.

**Time:** about 1–2 hours. Use any tools you like, including AI. In the interview
you will walk us through your changes and why you made them, and then we will talk
about how this would work at a much larger scale.

## How it works today

The example below follows one traveller in Germany who opens the app.

```mermaid
sequenceDiagram
    autonumber

    actor T as Traveller in Germany

    box rgb(230,240,255) Partner Management Team
        participant D as Deal Settings
    end

    box rgb(235,250,235) Offers Team
        participant F as Deal Feed
        participant C as Budget Check
    end

    box rgb(255,245,230) Platform Team
        participant S as Shared Counter Service
    end

    participant P as Skyline Air (Partner)

    Note over D,P: A deal is an offer from a travel company, such as a cheap flight.<br/>The travel company is the partner. The partner pays us for each booking<br/>and sets a daily budget — the maximum amount they will pay in one day.

    T->>F: Opens the app
    F->>D: Get deals available in Germany
    D-->>F: Returns 6 deals, including<br/>Skyline Air: Berlin → Lisbon

    Note over D: Skyline Air pays $4 per booking normally<br/>and $5 when the traveller is in Germany.<br/>Daily budget: $100.

    F->>C: Check which deals are still within budget

    loop For each of the 6 deals
        C->>S: How much has this deal spent<br/>and how many bookings has it made today?
        C->>S: How much has this partner spent today?
    end

    Note over C,S: This means 30 requests for a single app open.<br/>The app runs on 4 servers, all using the same shared service.<br/>Each request is a slow network call.<br/><br/>If the service does not respond, the current logic<br/>keeps the deal instead of blocking it.

    S-->>C: Berlin → Lisbon: 20 bookings today<br/>$80 confirmed spend

    Note over S,P: Skyline Air has confirmed 16 of the 20 bookings so far.<br/>The other 4 bookings are valid, but their confirmations<br/>arrive a few minutes later.

    Note over C: The current check only counts confirmed spend: $80.<br/><br/>It also calculates a backup estimate:<br/>20 bookings × $4 = $80.<br/>It uses the standard $4 price instead of the German $5 price.<br/>The estimate is only trusted if it exceeds $150<br/>(1.5 × the $100 daily budget).<br/><br/>Both checks say the deal is within budget.

    C-->>F: Berlin → Lisbon is within budget
    F-->>T: Shows the deal

    T->>F: Books the flight
    F->>S: Records 1 new booking

    P->>S: Confirms the 4 pending bookings<br/>and the new booking

    Note over S,P: Actual spend is now $80 + (5 × $5) = $105.<br/>The deal has exceeded its $100 daily budget.<br/>Skyline Air will not pay for the extra booking.

    Note over T,P: Business requirement:<br/><br/>• Count every booking, including unconfirmed bookings.<br/>• Use the actual price for the traveller's country.<br/>• For Germany, the calculation should be $80 + (4 × $5) = $100.<br/>• The deal should therefore have been hidden before the booking.<br/>• Ask the shared counter service at most once per app open.<br/>• If the service is unavailable, do not risk exceeding the budget.<br/>• At the same time, keep showing deals that are clearly affordable.
```

Where the 30 requests come from: the budget check asks four questions for each
deal (money confirmed and number of bookings, both for today and in total), and one
more question about the partner. Six deals times five questions is 30.

## Who owns what

| In the diagram | Folder | Team | What it does |
|---|---|---|---|
| Deal Settings | `deals/` | Partner Management | Stores each deal's price, its price in each country, and its budgets. Also stores each partner's daily limit for all its deals together. |
| Deal Feed | `feed/feed.go` | Offers (you) | Builds the list of deals a traveller sees. |
| Budget Check | `feed/checks.go` | Offers (you) | Leaves out deals whose budget is used up. **This is the code to improve.** |
| Shared Counter Service | `stats/` | Platform | Remembers bookings and confirmed money. All app servers share it. Every question to it is a slow network call, and sometimes it does not answer. |
| — | `cmd/sim/` | — | Plays one busy day and checks the rules below. |

You may change any file if you need to, but be ready to explain why.

## Make these four rules true

1. Each time a traveller opens the app, the shared counter service is asked at most once.
2. Opening the app is fast: 99 out of 100 app opens take 15 ms or less.
3. No deal, and no partner, goes more than 5% over its daily budget, even when
   confirmations arrive late and the counter service sometimes does not answer.
4. Travellers always see the deals that can still be afforded.

## Run it

You only need Go 1.24 or newer. No Docker and no network are needed.

```sh
cd tripmiles
go test ./...        # the rules as small tests
go run ./cmd/sim     # one busy day; prints PASS or FAIL for each rule
```

Today, three tests fail, and the simulator shows three of the four rules failing.
You are done when every test passes and the simulator prints PASS four times.
Please also run `go test -race ./...`.
