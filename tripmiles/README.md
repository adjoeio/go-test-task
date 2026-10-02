# TripMiles: pre-interview task

TripMiles is a small app that shows travel deals to travellers.
You are on the **Offers team**. Your team builds the list of deals that a traveller sees.

The app works. But under real traffic, three of its four rules fail.
Your task is to change the code so that all four rules pass.

- **Time:** about 1 to 2 hours.
- **Tools:** use any tools that you like, including AI.
- **Interview:** you explain your changes and why you made them.
  Then we talk about how the same design works at a much larger scale.

## Words used in this task

- **Partner:** a travel company, for example an airline. Skyline Air is a partner.
- **Deal:** an offer from a partner, for example a cheap flight.
- **Price:** the amount that the partner pays us for one booking.
  Some countries have a different price. For example, travellers in Germany cost $5, not $4.
- **Daily budget:** the most that the partner pays for one deal in one day.
  When the budget is used, the deal must not be on the list.
- **Partner daily limit:** the most that a partner pays for all its deals together in one day.
- **Confirmation:** a message from the partner that says a booking is valid.
  A confirmation arrives a few minutes after the booking.
- **Confirmed spend:** the money from bookings that the partner has confirmed.
- **Unconfirmed booking:** a valid booking whose confirmation has not arrived yet.
- **Shared counter service:** the service that counts bookings and confirmed spend.
  All 4 app servers use this one service.
- **Request:** one call from an app server to the shared counter service.
  Each request goes over the network, so each request is slow.
  Sometimes the service does not answer a request.

## How it works today

The diagram follows one traveller in Germany who opens the app.

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

The diagram shows 30 requests for one app open. They come from two checks:

- The **deal budget check** sends 4 requests for each deal:
  confirmed spend today, bookings today, confirmed spend in total, and bookings in total.
- The **partner limit check** sends 1 request for each deal.
- 6 deals × 5 requests = 30 requests.

## Who owns what

| Name in the diagram | Folder | Team | What it does |
|---|---|---|---|
| Deal Settings | `deals/` | Partner Management | Stores the price, the country prices and the budgets of each deal. Stores the partner daily limit of each partner. |
| Deal Feed | `feed/feed.go` | Offers (your team) | Builds the list of deals for one app open. |
| Budget Check | `feed/checks.go` | Offers (your team) | Contains the deal budget check and the partner limit check. **Change this file.** |
| Shared Counter Service | `stats/` | Platform | Counts bookings and confirmed spend. Each request is slow. Sometimes it does not answer. |
| (not in the diagram) | `cmd/sim/` | none | Runs one busy day on 4 servers. Then it checks the four rules. |

You can change other files too. If you do, be ready to explain why.

## The four rules

1. Each app open sends at most 1 request to the shared counter service.
2. 99 of 100 app opens take 15 ms or less.
3. No deal and no partner spends more than 105% of its budget.
   This must be true when confirmations arrive late.
   It must also be true when the shared counter service does not answer.
4. Affordable deals stay on the list.

## Run it

You need Go 1.24 or newer. You do not need Docker or a network connection.

1. Go to the task folder:
   ```sh
   cd tripmiles
   ```
2. Run the tests:
   ```sh
   go test ./...
   ```
3. Run the simulated day:
   ```sh
   go run ./cmd/sim
   ```
4. Run the tests with the race detector:
   ```sh
   go test -race ./...
   ```

Today, 3 tests fail and 3 tests pass. The simulator shows `FAIL` for rules 1, 2 and 3.

## You are done when

- [ ] `go test ./...` shows no failures.
- [ ] `go run ./cmd/sim` shows `PASS` for all four rules.
- [ ] `go test -race ./...` shows no data races.
- [ ] You can explain each change that you made.
