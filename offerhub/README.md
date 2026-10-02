# OfferHub — pre-interview task

OfferHub is a small offerwall backend. Users open a list of game offers, see
how many **coins** they will get for reaching milestones in each game, install
one, and get paid when they reach a milestone. Advertisers pay us per install.

You are on the **Offers Core** team. Two functions are not written yet.

**Time: about 1–2 hours. Use any tools you like, including AI.**
We care that you understand *why* your solution works; we will build on it
together in the interview.

## Run it

Needs Go 1.24+. No Docker, no network.

```sh
cd offerhub
go test ./...        # the rules as tests
go run ./cmd/sim     # a busy hour of traffic; prints one CHECK line per rule
```

Done = both pass. Also try `go test -race ./...`.

## The rules

1. **Coins come from the advertiser's price.**
   `coins = CPI × (1 − margin) × coins per USD` — already written in `coins/`.
2. **What we show is a promise.** When a user is shown an offer we store a
   promise of coins per milestone (`promise/`, already written). Advertisers
   can change their campaign at any time; that must not change what an
   existing user is paid.
3. **Pay each milestone exactly once.** Milestone events come from a queue that
   delivers *at least once*: the same event can arrive twice, at the same time,
   on different server instances. The instances share only the KV store
   (`platform/`).
4. **Never spend more than a campaign's budget.** Spend is kept in a shared
   store that is sometimes unavailable, and every lookup is a network call.

## What to write

| File | Function | Rules |
|---|---|---|
| `reward/reward.go` | `Pay` | 2, 3 |
| `budget/budget.go` | `Filter` | 4 |

Everything else is there to help you. Change it if you need to, but be ready to
explain why.

## Who owns what

```
catalog/   Campaign Management team   advertisers' campaign settings
platform/  Platform team              KV store, feature flags, clock
offers/    Offers Core (you)          builds the offer list
promise/   Offers Core (you)          what we promised each user
reward/    Offers Core (you)          pays users
budget/    Offers Core (you)          campaign spend
cmd/sim/   —                          traffic simulator
```

## In the interview (45 min)

1. You walk us through your solution (about 10 min).
2. We extend it together, live, AI allowed (about 20 min).
3. We talk about shipping it safely: rollout, monitoring, what "done" means.

No need to prepare anything else.
