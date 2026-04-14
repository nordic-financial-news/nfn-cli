---
name: nfn-cli
description: |
  Use the nfn CLI to query Nordic financial news — articles, stories, companies, exchanges,
  indices, watchlists, and categories across Sweden, Denmark, Norway, Finland, and Iceland.
  ALWAYS use this skill when the user wants to: find news about a Nordic company (Ericsson,
  Volvo, Novo Nordisk, Northvolt, Maersk, etc.), look up a company by ticker or ID (VOLV-B,
  ERIC-B, NOVO-B, NDA-SE, MAERSK-B), list companies on an exchange (XSTO, XCSE, XHEL, XOSL), see index
  constituents (OMXS30, OMXC25, OMXH25, OBX), search for financial news by topic, filter
  articles by country or category, manage watchlists, or script/automate news queries with
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

8. **Country codes are ISO2** — use `SE` for Sweden, `DK` for Denmark, `FI` for Finland, `NO` for Norway, `IS` for Iceland. Run `nfn countries list` if unsure.

9. **Exchange codes are MIC codes** — use `XSTO` for Stockholm, `XCSE` for Copenhagen, `XHEL` for Helsinki, `XOSL` for Oslo. Run `nfn exchanges list` if unsure.

10. **Disambiguate company names before assuming an identifier** — many Nordic company names map to multiple entities. For example, "Volvo" could mean AB Volvo (Volvo Group, ticker VOLV-B), Volvo Car AB (VOLCAR-B), Volvo Lastvagnar Sverige Ab, or Volvo Penta. When the user mentions a company by name, search first with `nfn companies list --q "name"` to see all matches, then either ask the user which one they mean or pick the most likely match and mention the alternatives. Never silently assume an identifier from an ambiguous name. Note: not all companies have a ticker — use the company ID when no ticker is available.

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

### Filtering Fields

```bash
nfn articles list --fields "id,title,published_at"  # Return only specified fields (JSON mode)
```

## Quick Reference

| Task | Command |
|------|---------|
| **Articles** | |
| List articles | `nfn articles list --format json` |
| Filter by country | `nfn articles list --country SE` |
| Filter by category | `nfn articles list --category "Economic Policy"` |
| Filter by ticker | `nfn articles list --ticker VOLV-B` |
| Filter by source | `nfn articles list --sources <id1,id2>` |
| Filter by date range | `nfn articles list --published-after 2025-01-01 --published-before 2025-06-01` |
| Search articles by text | `nfn articles list --q "battery"` |
| Only listed companies | `nfn articles list --listed` |
| Only watchlisted companies | `nfn articles list --watchlist` |
| Filter by content type | `nfn articles list --content-type "press release"` |
| Fetch specific IDs | `nfn articles list --ids "id1,id2,id3" --format json` |
| Incremental sync | `nfn articles list --updated-after "2025-03-01T00:00:00Z" --all --format json` |
| Get article details | `nfn articles get <id> --format json` |
| **Stories** | |
| List stories | `nfn stories list --format json` |
| Filter stories by country | `nfn stories list --country SE` |
| Filter stories by category | `nfn stories list --category "Mergers & Acquisitions"` |
| Filter stories by ticker | `nfn stories list --ticker VOLV-B` |
| Filter stories by source | `nfn stories list --sources <id1,id2>` |
| Only listed companies | `nfn stories list --listed` |
| Only watchlisted companies | `nfn stories list --watchlist` |
| Get story details | `nfn stories get <id> --format json` |
| **Companies** | |
| List companies | `nfn companies list --format json` |
| Search companies | `nfn companies list --q "Volvo"` |
| Filter by exchange | `nfn companies list --exchange XSTO` |
| Filter by sector | `nfn companies list --sector Financials` |
| Only listed companies | `nfn companies list --listed` |
| Only active companies | `nfn companies list --is-active` |
| Get company details | `nfn companies get VOLV-B --format json` |
| Company articles | `nfn companies articles VOLV-B --limit 5` |
| Primary articles only | `nfn companies articles VOLV-B --primary-only` |
| Company stories | `nfn companies stories VOLV-B --limit 5` |
| **Exchanges** | |
| List exchanges | `nfn exchanges list --format json` |
| Filter by country | `nfn exchanges list --country SE` |
| Get exchange details | `nfn exchanges get XSTO --format json` |
| List exchange companies | `nfn exchanges companies XSTO --limit 20` |
| **Indices** | |
| List indices | `nfn indices list --format json` |
| Filter by exchange | `nfn indices list --exchange XSTO` |
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
├── Know the company name but not identifier? → nfn companies list --q "Volvo"
│   └── Multiple matches? → Ask user or note which entity you're using
├── Know the topic? → nfn search "topic" --type articles
├── Want a specific country? → nfn articles list --country SE
├── Want a specific source? → nfn sources list (to see IDs)
│   └── then → nfn articles list --sources <id1,id2>
├── Want a specific category? → nfn categories (to see options)
│   └── then → nfn articles list --category "Economic Policy"
├── Want stories (clustered articles)? → nfn stories list
├── Only about listed companies? → nfn articles list --listed
├── Only about watchlisted companies? → nfn articles list --watchlist
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
├── Want companies in a country? → nfn companies list --country SE
├── Only active companies? → nfn companies list --is-active
└── Cross-resource search? → nfn search "query" --type companies
```

### Exploring Markets

```
Exploring Nordic markets?
├── Which exchanges exist? → nfn exchanges list
├── Details on an exchange? → nfn exchanges get XSTO
├── Which indices exist? → nfn indices list
├── Which indices on an exchange? → nfn indices list --exchange XSTO
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
