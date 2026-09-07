---
name: nfn-cli
description: |
  Use the nfn CLI to query Nordic financial news — articles, stories, companies, exchanges,
  indices, watchlists, calendar events, and categories across Sweden, Denmark, Norway, Finland, and Iceland.
  ALWAYS use this skill when the user wants to: find news about a Nordic company (Ericsson,
  Volvo, Novo Nordisk, Northvolt, Maersk, etc.), look up a company by ticker or ID (VOLV-B,
  ERIC-B, NOVO-B, NDA-SE, MAERSK-B), list companies on an exchange (XSTO, XCSE, XHEL, XOSL), see index
  constituents (OMXS30, OMXC25, OMXH25, OBX), search for financial news by topic, filter
  articles by country or category, manage watchlists, see upcoming financial calendar events
  (earnings reports, dividends, AGMs), or script/automate news queries with
  JSON output. Also use when troubleshooting nfn CLI errors (401, auth issues, rate limits)
  or when the user mentions nordicfinancialnews.com. Even if the user doesn't say "nfn"
  explicitly, trigger this skill whenever Nordic financial news data is needed.
triggers:
  # Direct invocations
  - nfn
  - /nfn
  # Resource types
  - nfn articles
  - nfn stories
  - nfn companies
  - nfn exchanges
  - nfn indices
  - nfn watchlists
  - nfn calendar
  - nfn categories
  - nfn countries
  - nfn sources
  - nfn search
  # Common actions
  - nordic financial news
  - nordic news
  - financial news nordic
  - scandinavian news
  - scandinavian stocks
  - stock news
  - company news
  - search news
  - find company
  - look up ticker
  - look up company
  - latest articles
  - latest stories
  - market news
  # Questions
  - what's the news on
  - any news about
  - news for ticker
  - articles about
  - stories about
  # Market exploration
  - stockholm exchange
  - copenhagen exchange
  - helsinki exchange
  - oslo exchange
  - nordic exchange
  - OMXS30
  - OMXC25
  - OMXH25
  - OBX
  # Tickers (examples)
  - VOLV-B
  - ERIC-B
  - NOVO-B
  - NDA-SE
  - MAERSK-B
  # Domain
  - nordicfinancialnews.com
invocable: true
argument-hint: "[command] [args...]"
---

# /nfn - Nordic Financial News CLI

Query Nordic financial news articles, stories, companies, stock exchanges, indices, watchlists, and categories from the terminal.

## Agent Invariants

**Follow these rules when using the nfn CLI:**

1. **Choose the right output format** — use `--format json` when you need to extract or process data programmatically; use `--format table` (or omit the flag in a TTY) when presenting results to a human. The CLI auto-detects: table in interactive terminals, JSON when piped. Always pass `--format json` explicitly when running from scripts or agents to avoid ambiguity.

2. **Use short ticker format** — tickers are written as `VOLV-B`, not `VOLV-B.ST` or `VOLV-B:XSTO`. The exchange suffix is never part of the ticker in this system.

3. **Authenticate before querying** — most commands require an API key. Use `nfn auth login` (interactive, masked input) or set the `NFN_API_KEY` environment variable. Never ask users to paste API keys in plaintext. Use `nfn auth status` to check authentication and rate limits.

4. **Introspect with `nfn commands`** — outputs the full command catalog as structured JSON, including flags, arguments, examples, and related commands. Useful when you're unsure about available flags for a specific command. Filter with `--command "articles list"` for a single command.

5. **Diagnose with `nfn doctor`** — runs health checks on config, API connectivity, authentication, and CLI version. Use this first when something isn't working.

6. **JSON output uses an envelope** — all JSON responses wrap data in `{ok, data, summary, breadcrumbs}`. Errors return `{ok: false, error, data: null}`. Access the actual payload via `.data`.

7. **Breadcrumbs suggest next commands** — the `breadcrumbs` array in JSON output suggests logical follow-up commands. Use these to navigate the data model (e.g., from a company to its articles).

8. **Country codes are ISO2** — use `SE` for Sweden, `DK` for Denmark, `FI` for Finland, `NO` for Norway, `IS` for Iceland. Run `nfn countries list` if unsure. Every country filter accepts a comma-separated list to combine countries, e.g. `--country SE,NO,FI`.

9. **Pick the right country filter: `--country`, `--market`, or `--domicile`** — these ask three different questions, and the one you want depends on what "Swedish" means in the request. All three are ISO2 and comma-separated, and they combine as an intersection with each other and with every other filter.

   - `--market` — **companies with an active equity listing on an exchange in these countries.** This is the "on the Swedish market" question. `--market SE` covers Nasdaq Stockholm, First North, Spotlight, and NGM together, so it reaches a company listed only on a growth-market segment that `--exchange XSTO` would miss. It includes foreign-domiciled issuers that are listed locally.
   - `--domicile` — **companies legally domiciled in these countries.** This is the "Swedish company" question. It reaches unlisted companies, and it excludes foreign-domiciled issuers that only happen to be listed locally.
   - `--country` — resource-dependent, so read it carefully:
     - On `articles` and `stories`, this is the item's own country.
     - On `calendar`, this is the event's own country (the issuer's, falling back to the source's) — distinct from both `--market` and `--domicile`.
     - On `companies`, this already means legal domicile; `--domicile` is the canonical name for the same filter. Passing both applies both, intersected, like any other filter pair.

   Available on `articles list`, `stories list`, `companies list`, and `calendar list`.

10. **Exchange codes are MIC codes** — use `XSTO` for Stockholm, `XCSE` for Copenhagen, `XHEL` for Helsinki, `XOSL` for Oslo. Run `nfn exchanges list` if unsure. `--exchange` accepts a comma-separated list of MICs to cover several venues in one call, e.g. `--exchange XSTO,XCSE`. Reach for `--market SE` instead when you want a whole national market rather than named venues. Note that `--exchange` matches only active listings (and, on `nfn indices list`, only active exchanges) — a company whose sole listing on that exchange is delisted will not appear.

11. **Disambiguate company names before assuming an identifier** — many Nordic company names map to multiple entities. For example, "Volvo" could mean AB Volvo (Volvo Group, ticker VOLV-B), Volvo Car AB (VOLCAR-B), Volvo Lastvagnar Sverige Ab, or Volvo Penta. When the user mentions a company by name, search first with `nfn companies list --q "name"` to see all matches, then either ask the user which one they mean or pick the most likely match and mention the alternatives. Never silently assume an identifier from an ambiguous name. Note: not all companies have a ticker — use the company ID when no ticker is available. To filter articles, stories, or calendar events by such a company, use `--company <id>` (the `--ticker` filter only reaches listed companies; `--company` also reaches private, unlisted, and delisted ones). Both `--ticker` and `--company` accept a comma-separated list (max 25, matching any of the named companies), e.g. `--ticker VOLV-B,ERIC-B`. Every value must resolve: an unknown ticker or ID now fails the whole request with a `400` naming the offending value (e.g. `Unknown ticker(s): VOLVB`) rather than silently returning an empty list. The other company-selecting filters (`--exchange`, `--market`, `--domicile`) still narrow silently on an unknown code. To resolve a batch of ids or tickers back to company names in one call — e.g. after a calendar or article page hands you `company_id`s — use `nfn companies list --company <id1,id2>` or `nfn companies list --ticker VOLV-B,ERIC-B` (both exact filters, max 25, and paginate normally). Results come back in name order, so key each row by its own `id`; a shared ticker can resolve to more than one company.

12. **Article title, summary, and key points are machine-translated and AI-paraphrased** — the `title`, `summary`, and `key_points` fields are machine-translated from the source language and AI-paraphrased. They are not verbatim quotations and are not human-verified. When accuracy matters, follow `article_url` to the original source and attribute accordingly.

13. **Calendar rows carry the issuer inline** — each calendar event has `company_id` plus, by default, an inline `company` object (`id`, `name`, `ticker`), so you don't need a separate `nfn companies get` per row to render the issuer. The table's Company column shows the name (falling back to the id if the issuer is withheld); read `data.calendar_events[].company` in JSON for the full object. `ticker` is `null` for an unlisted issuer. Note that `--fields` *narrows* the response to `id` plus the fields you name, so `--fields company` returns only the issuer and drops title/date/etc. — don't use it just to get the name, which is already there.

14. **Calendar date windows are issuer-local when you pass a bare date** — on `nfn calendar list`, `--scheduled-after` / `--scheduled-before` read a bare date (`2026-10-01`) as that calendar date in the *issuer's own* timezone, inclusive at both ends, so `--scheduled-after 2026-10-01 --scheduled-before 2026-10-31` returns October for every issuer regardless of where they sit relative to UTC. Day-precision events (earnings dates, AGMs, dividends given only as a date) are stored at the issuer's local midnight, so this is what you want in almost every case — do not widen the window by a day and filter on `local_time` yourself. Pass a full ISO 8601 datetime (`2026-11-01T00:00:00Z`, or with a UTC offset like `+02:00`) only when you genuinely mean an exact instant: `--scheduled-after` is then at-or-after and `--scheduled-before` is *strictly before* that instant. A datetime is **not** a shortcut to an exclusive calendar end — `--scheduled-before 2026-07-01T00:00:00Z` still picks up a 1 July event for an issuer ahead of UTC, because its local midnight (`2026-06-30T22:00:00Z` in Sweden) falls before the UTC one. For a window that means calendar days, use the bare-date form.

15. **Check `plan_limited` before treating a result as complete** — list responses include `data.plan_limited: true` when the API key's plan hid records from the result. A short result is not proof that few records exist. When it's true, say so rather than reporting the visible count as the full picture; the fix is a plan upgrade, not a different query or a larger `--limit`. The key is omitted entirely when no restriction applied, so test for `true` (`.data.plan_limited == true`), not merely for presence. In table mode this surfaces as a note under the table.

### Output Modes

| Goal | Flag | What you get |
|------|------|-------------|
| Human-readable tables | `--format table` (default in TTY) | Colored tables with key columns |
| Structured data for processing | `--format json` (default when piped) | JSON envelope: `{ok, data, summary, breadcrumbs}` |
| Disable color | `--no-color` | Plain text tables without ANSI codes |

### Pagination

```bash
nfn <cmd> list --limit 10       # Cap results per page
nfn <cmd> list --all            # Fetch all pages (may be slow for large datasets)
nfn <cmd> list --cursor <token> # Continue from a specific page
```

`--all` fetches every page automatically. Use `--limit` for a quick sample. `--cursor` is for manual pagination using the `next_cursor` value from a previous response.

### Plan Limits

Some plans return only a subset of the matching records. When that happens the API sets a top-level flag that the CLI passes through:

```bash
nfn stories list --limit 25 --format json | jq '.data.plan_limited'   # true when records were hidden
```

```json
{
  "ok": true,
  "data": { "stories": [ ... ], "plan_limited": true },
  "summary": "4 stories (limited by your plan — some records were hidden)"
}
```

This is independent of pagination: `plan_limited` can be `true` even when `next_cursor` is null, and `--all` still reports it after following every page. The key is absent when nothing was restricted, so `true` means capped and absent means complete.

### Filtering Fields

```bash
nfn articles list --fields "id,title,published_at"  # Return only specified fields (JSON mode)
```

## Quick Reference

| Task | Command |
|------|---------|
| **Articles** | |
| List articles | `nfn articles list --format json` |
| Filter by country | `nfn articles list --country SE,NO` |
| Filter by category | `nfn articles list --category "Economic Policy"` |
| Filter by ticker | `nfn articles list --ticker VOLV-B` |
| Filter by several tickers | `nfn articles list --ticker VOLV-B,ERIC-B` |
| Filter by company ID | `nfn articles list --company <id>` |
| Only primary-company articles | `nfn articles list --ticker VOLV-B --primary-only` |
| Filter by exchange | `nfn articles list --exchange XSTO,XCSE` |
| Filter by listing market | `nfn articles list --market SE` |
| Filter by legal domicile | `nfn articles list --domicile FI` |
| Filter by index | `nfn articles list --index OMXS30` |
| Filter by sector | `nfn articles list --sector Industrials` |
| Filter by source | `nfn articles list --sources <id1,id2>` |
| Filter by source type | `nfn articles list --source-type stock_exchange,wire_service` |
| Filter by date range | `nfn articles list --published-after 2025-01-01 --published-before 2025-06-01` |
| Search articles by text | `nfn articles list --q "battery"` |
| Search, newest-first, all pages | `nfn articles list --q "battery" --sort latest --all` |
| Only listed companies | `nfn articles list --listed` |
| Filter by watchlist | `nfn articles list --watchlist-id <id>` |
| Filter by content type | `nfn articles list --content-type "press release"` |
| Fetch specific IDs | `nfn articles list --ids "id1,id2,id3" --format json` |
| Incremental sync | `nfn articles list --updated-after "2025-03-01T00:00:00Z" --all --format json` |
| Get article details | `nfn articles get <id> --format json` |
| **Stories** | |
| List stories | `nfn stories list --format json` |
| Filter stories by country | `nfn stories list --country SE,NO` |
| Filter stories by category | `nfn stories list --category "Mergers & Acquisitions"` |
| Search stories by text | `nfn stories list --q "battery"` |
| Search, newest-first, all pages | `nfn stories list --q "battery" --sort latest --all` |
| Filter stories by date range | `nfn stories list --published-after 2026-01-01 --published-before 2026-06-01` |
| Filter stories by ticker | `nfn stories list --ticker VOLV-B` |
| Filter stories by several tickers | `nfn stories list --ticker VOLV-B,ERIC-B` |
| Filter stories by company ID | `nfn stories list --company <id>` |
| Filter stories by exchange | `nfn stories list --exchange XSTO,XCSE` |
| Filter stories by listing market | `nfn stories list --market SE` |
| Filter stories by legal domicile | `nfn stories list --domicile FI` |
| Filter stories by index | `nfn stories list --index OMXS30` |
| Filter stories by sector | `nfn stories list --sector Industrials` |
| Filter stories by source | `nfn stories list --sources <id1,id2>` |
| Only listed companies | `nfn stories list --listed` |
| Filter by watchlist | `nfn stories list --watchlist-id <id>` |
| Get story details | `nfn stories get <id> --format json` |
| **Calendar** | |
| List calendar events | `nfn calendar list --format json` |
| Filter by ticker | `nfn calendar list --ticker VOLV-B` |
| Filter by company ID | `nfn calendar list --company <id>` |
| Filter by event type | `nfn calendar list --event-type earnings_report,dividend` |
| Filter by status | `nfn calendar list --status scheduled` |
| Filter by date range (issuer-local, inclusive) | `nfn calendar list --scheduled-after 2026-01-01 --scheduled-before 2026-06-30` |
| Cut at an exact UTC instant (not a local calendar day) | `nfn calendar list --scheduled-before 2026-07-01T00:00:00Z` |
| Filter by event country | `nfn calendar list --country SE,NO` |
| Filter by listing market | `nfn calendar list --market SE` |
| Filter by legal domicile | `nfn calendar list --domicile FI` |
| Filter by exchange | `nfn calendar list --exchange XSTO,XCSE` |
| Filter by watchlist | `nfn calendar list --watchlist-id <id>` |
| Get calendar event details | `nfn calendar get <id> --format json` |
| **Companies** | |
| List companies | `nfn companies list --format json` |
| Search companies | `nfn companies list --q "Volvo"` |
| Batch lookup by ticker | `nfn companies list --ticker VOLV-B,ERIC-B` |
| Batch lookup by ID | `nfn companies list --company <id1,id2>` |
| Filter by exchange | `nfn companies list --exchange XSTO,XCSE` |
| Filter by listing market | `nfn companies list --market SE` |
| Filter by legal domicile | `nfn companies list --domicile SE` |
| Filter by sector | `nfn companies list --sector Financials` |
| Only companies with an active listing | `nfn companies list --listed` |
| Only active companies | `nfn companies list --is-active` |
| Get company details | `nfn companies get VOLV-B --format json` |
| Company articles | `nfn companies articles VOLV-B --limit 5` |
| Primary articles only | `nfn companies articles VOLV-B --primary-only` |
| Company articles, filtered | `nfn companies articles VOLV-B --category "Mergers & Acquisitions" --published-after 2026-01-01` |
| Company articles, search | `nfn companies articles VOLV-B --q "battery" --sort latest --all` |
| Company stories | `nfn companies stories VOLV-B --limit 5` |
| Company stories, filtered | `nfn companies stories VOLV-B --category "Mergers & Acquisitions" --published-after 2026-01-01` |
| **Exchanges** | |
| List exchanges | `nfn exchanges list --format json` |
| Filter by country | `nfn exchanges list --country SE` |
| Get exchange details | `nfn exchanges get XSTO --format json` |
| List exchange companies | `nfn exchanges companies XSTO --limit 20` |
| **Indices** | |
| List indices | `nfn indices list --format json` |
| Filter by exchange | `nfn indices list --exchange XSTO,XCSE` |
| Get index details | `nfn indices get OMXS30 --format json` |
| List index companies | `nfn indices companies OMXS30 --all` |
| **Search** | |
| Search everything | `nfn search "Volvo" --format json` |
| Search specific type | `nfn search "IPO" --type articles` |
| Limit results | `nfn search "battery" --limit 5` |
| **Categories** | |
| List categories | `nfn categories --format json` |
| **Countries** | |
| List countries | `nfn countries list` |
| Get country details | `nfn countries get SE` |
| **Sources** | |
| List sources | `nfn sources list --format json` |
| Paginate all sources | `nfn sources list --all --format json` |
| Filter articles by source | `nfn articles list --sources <id1,id2>` |
| Filter articles by source type | `nfn articles list --source-type stock_exchange` |
| Filter stories by source | `nfn stories list --sources <id1,id2>` |
| **Watchlists** | |
| List watchlists | `nfn watchlists list --format json` |
| Get watchlist + companies | `nfn watchlists get <id> --format json` |
| **Auth & Diagnostics** | |
| Log in | `nfn auth login` |
| Check auth + rate limits | `nfn auth status` |
| Log out | `nfn auth logout` |
| Run diagnostics | `nfn doctor` |
| Show CLI version | `nfn version` |
| Command catalog (JSON) | `nfn commands --format json` |
| Single command help | `nfn commands --command "articles list" --format json` |

## Decision Trees

### Finding News

```
Looking for news?
├── Know the ticker or company ID? → nfn companies articles VOLV-B
│   ├── Narrow that company's own feed? → nfn companies articles VOLV-B --category "M&A" --published-after 2026-01-01
│   ├── Search within it? → nfn companies articles VOLV-B --q "battery" --sort latest
│   └── Drop passing mentions? → nfn companies articles VOLV-B --primary-only
├── Know the company name but not identifier? → nfn companies list --q "Volvo"
│   └── Multiple matches? → Ask user or note which entity you're using
├── Several companies at once? → nfn articles list --ticker VOLV-B,ERIC-B
├── Know the topic? → nfn search "topic" --type articles
├── Want a specific country? → nfn articles list --country SE
├── Want a whole national market (all its venues)? → nfn articles list --market SE
├── Want companies legally domiciled somewhere? → nfn articles list --domicile SE
├── Want a specific source? → nfn sources list (to see IDs)
│   └── then → nfn articles list --sources <id1,id2>
├── Want a kind of source (exchange feeds, wire services...)? → nfn articles list --source-type stock_exchange
│   └── Fastest disclosure tier → nfn articles list --source-type stock_exchange
├── Want a specific category? → nfn categories (to see options)
│   └── then → nfn articles list --category "Economic Policy"
├── Want stories (clustered articles)? → nfn stories list
├── Want upcoming events (earnings, dividends, AGMs)? → nfn calendar list --ticker VOLV-B
├── Only about listed companies? → nfn articles list --listed
├── Only about companies in a watchlist? → nfn watchlists list (to see IDs)
│   └── then → nfn articles list --watchlist-id <id>
├── Have an article ID? → nfn articles get <id>
└── Not sure what's available? → nfn search "query"
```

### Finding Companies

```
Looking for a company?
├── Know the ticker or company ID? → nfn companies get VOLV-B
├── Know part of the name? → nfn companies list --q "Volvo"
│   └── Multiple matches? → Show user the options, ask which one they mean
├── Want companies on an exchange? → nfn exchanges companies XSTO
├── Want companies in an index? → nfn indices companies OMXS30
├── Want companies in a sector? → nfn companies list --sector Financials
├── Want companies in a country? → nfn companies list --domicile SE
├── Want companies traded on a national market? → nfn companies list --market SE
│   └── Includes First North / Spotlight / NGM, and foreign issuers listed there
├── Want companies on named venues? → nfn companies list --exchange XSTO,XCSE
├── Only active companies? → nfn companies list --is-active
└── Cross-resource search? → nfn search "query" --type companies
```

### Exploring Markets

```
Exploring Nordic markets?
├── Which exchanges exist? → nfn exchanges list
├── Details on an exchange? → nfn exchanges get XSTO
├── Which indices exist? → nfn indices list
├── Which indices on an exchange? → nfn indices list --exchange XSTO,XCSE
├── Companies in an index? → nfn indices companies OMXS30
├── Which countries are covered? → nfn countries list
└── Country details + exchanges? → nfn countries get SE
```

### Troubleshooting

```
Something not working?
├── Run diagnostics → nfn doctor
├── Not authenticated? → nfn auth login
├── Check rate limits → nfn auth status
├── Check CLI version → nfn version
└── Need command help → nfn commands --command "articles list"
```

## Common Workflows

### Get company details with registry data

```bash
# Company details include active status, country, and Nordic business registry data
# (registered name, company form, city, founded, employees, industry, share capital, etc.)
nfn companies get VOLV-B --format json
```

### Get latest news for a company (by name)

```bash
# Step 1: Search for the company — names are often ambiguous
# "Volvo" returns 4 entities: AB Volvo (VOLV-B), Volvo Car AB (VOLCAR-B),
# Volvo Lastvagnar Sverige Ab, and Volvo Penta
nfn companies list --q "Volvo" --format json

# Step 2: Confirm which entity with the user, then fetch news by identifier (ticker or company ID)
nfn companies articles VOLV-B --limit 10 --format json

# Step 3: Optionally get story clusters for a higher-level view
nfn companies stories VOLV-B --limit 5 --format json
```

### Get latest news for a company (by identifier)

```bash
# When you already know the ticker or company ID, go directly
nfn companies articles VOLV-B --limit 10 --format json
nfn companies stories VOLV-B --limit 5 --format json

# The company feed takes the same filters as the flat feeds, scoped to that company:
# by category and date range...
nfn companies articles VOLV-B --category "Mergers & Acquisitions" --published-after 2026-01-01 --format json
# ...or a full-text search of the company's own coverage, newest first, all pages
nfn companies articles VOLV-B --q "battery" --sort latest --all --format json
# Drop articles where the company is only a passing mention
nfn companies articles VOLV-B --primary-only --format json
```

### Search for news on a topic

```bash
# Cross-resource search
nfn search "battery" --format json

# Or search just articles with text query
nfn articles list --q "battery" --limit 10 --format json
```

### Explore an exchange

```bash
# List all exchanges
nfn exchanges list

# Get details for Stockholm
nfn exchanges get XSTO --format json

# List companies on the exchange
nfn exchanges companies XSTO --all --format json
```

### Target a national market, a venue, or a domicile

```bash
# Everything traded on the Swedish market: Nasdaq Stockholm, First North,
# Spotlight and NGM together, including foreign issuers listed there
nfn companies list --market SE --all --format json

# Only the named venues you list, and only where the listing is still active
nfn companies list --exchange XSTO,XCSE --all --format json

# Swedish-domiciled companies, including private and unlisted ones
nfn companies list --domicile SE --all --format json

# The same distinction applies to content: news about issuers on the
# Swedish market, whatever country the article itself came from
nfn articles list --market SE --published-after 2026-01-01 --format json

# Earnings dates for issuers listed in Sweden or Denmark
nfn calendar list --market SE,DK --event-type earnings_report --format json
```

### Filter news by source

```bash
# List available sources and find the IDs you want
nfn sources list --all --format json

# Fetch articles from specific sources (comma-separated IDs, max 25)
nfn articles list --sources <source_id> --limit 10 --format json

# Combine source filter with other filters
nfn articles list --sources <id1>,<id2> --country SE --published-after 2026-01-01

# Same filter works on stories
nfn stories list --sources <source_id> --format json
```

### Filter articles by category and country

```bash
# See available categories
nfn categories

# Get Swedish articles in a specific category
nfn articles list --country SE --category "Mergers & Acquisitions" --limit 10
```

### Scripting with JSON output

```bash
# Pipe to jq for filtering
nfn articles list --ticker VOLV-B --format json | jq '.data.articles[] | {title, published_at}'

# Count articles per country
nfn articles list --all --format json | jq '[.data.articles[].country] | group_by(.) | map({country: .[0], count: length})'
```

### Check rate limits before a bulk operation

```bash
# See remaining quota
nfn auth status

# Then fetch all articles (uses pagination automatically)
nfn articles list --all --format json
```
