---
name: nfn-cli
description: |
  Use the nfn CLI to query Nordic financial news — articles, stories, events, companies, exchanges,
  indices, watchlists, calendar events, and categories across Sweden, Denmark, Norway, Finland, and Iceland.
  ALWAYS use this skill when the user wants to: find news about a Nordic company (Ericsson,
  Volvo, Novo Nordisk, Northvolt, Maersk, etc.), look up a company by ticker or ID (VOLV-B,
  ERIC-B, NOVO-B, NDA-SE, MAERSK-B), list companies on an exchange (XSTO, XCSE, XHEL, XOSL), see index
  constituents (OMXS30, OMXC25, OMXH25, OBX), search for financial news by topic, filter
  articles by country or category, manage watchlists, see what happened as grouped events
  rather than one row per article (earnings reports, acquisitions, executive changes),
  see upcoming financial calendar events
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
  - nfn events
  - nfn event-types
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
  - what happened
  - recent developments
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

Query Nordic financial news articles, stories, events, companies, stock exchanges, indices, watchlists, and categories from the terminal.

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

13. **Calendar rows carry the issuer inline** — each calendar event has `company_id` plus, by default, an inline `company` object (`id`, `name`, `ticker`), so you don't need a separate `nfn companies get` per row to render the issuer. The table's Company column shows the name (falling back to the id if the issuer is withheld); read `data.calendar_events[].company` in JSON for the full object. `ticker` is `null` for an unlisted issuer. Note that `--fields` *narrows* the result to `id` plus the fields you name, in the table as well as the JSON, so `--fields company` returns only the issuer and drops title/date/etc. Don't use it just to get the name, which is already there.

14. **Calendar date windows are issuer-local when you pass a bare date** — on `nfn calendar list`, `--scheduled-after` / `--scheduled-before` read a bare date (`2026-10-01`) as that calendar date in the *issuer's own* timezone, inclusive at both ends, so `--scheduled-after 2026-10-01 --scheduled-before 2026-10-31` returns October for every issuer regardless of where they sit relative to UTC. Day-precision events (earnings dates, AGMs, dividends given only as a date) are stored at the issuer's local midnight, so this is what you want in almost every case — do not widen the window by a day and filter on `local_time` yourself. Pass a full ISO 8601 datetime (`2026-11-01T00:00:00Z`, or with a UTC offset like `+02:00`) only when you genuinely mean an exact instant: `--scheduled-after` is then at-or-after and `--scheduled-before` is *strictly before* that instant. A datetime is **not** a shortcut to an exclusive calendar end — `--scheduled-before 2026-07-01T00:00:00Z` still picks up a 1 July event for an issuer ahead of UTC, because its local midnight (`2026-06-30T22:00:00Z` in Sweden) falls before the UTC one. For a window that means calendar days, use the bare-date form.

15. **Check `plan_limited` before treating a result as complete** — list responses include `data.plan_limited: true` when the API key's plan hid records from the result. A short result is not proof that few records exist. When it's true, say so rather than reporting the visible count as the full picture; the fix is a plan upgrade, not a different query or a larger `--limit`. The key is omitted entirely when no restriction applied, so test for `true` (`.data.plan_limited == true`), not merely for presence. In table mode this surfaces as a note under the table.

16. **Events group coverage; calendar events are scheduled dates — and their `--event-type` vocabularies are incompatible** — `nfn events list` returns *events*: one real-world development (an earnings report, an acquisition, an executive change) with the articles reporting it grouped under it. `nfn calendar list` returns *scheduled future dates* for one issuer. They answer different questions: "what happened" versus "what is coming up". Use events to collapse a pile of near-duplicate articles about one development into a single row, and to reach the whole story from any single article (`event_id` on `nfn articles get`) or story (`event_id` on `nfn stories get`).

    Their `--event-type` filters take **different, mutually rejected vocabularies**: events takes hyphenated Events-taxonomy slugs (`earnings-report`) from `nfn event-types`; calendar takes underscored types (`earnings_report`). Passing one to the other fails with a `400`. A few spellings overlap (e.g. `delisting`) while denoting different things, so copying a value across the two commands can quietly change what you asked for instead of erroring. Run `nfn event-types` to get the slug for `nfn events list`.

17. **The unfiltered events feed includes commentary and routine filings** — every event carries a `significance` tier: `notable` (material, front-page-worthy), `routine` (scheduled disclosures and filings), or `commentary` (opinion and analysis). `nfn events list` sends no default tier, mirroring the API, so what comes back mixes all three and opinion pieces sit alongside material news. When the user asks what is happening or what matters, pass `--significance notable`. Reach for `--sort significance` only to see all three tiers grouped in one pass — for a plain importance filter `--significance notable` with the default sort is better, since it returns an identical first page without the large ties inside each tier.

18. **Three different things make an events result incomplete** — `plan_limited` (invariant 15) means your plan hid records. `data.truncated: true`, which **only** the events feed sends and only under `--semantic`, means the search exhausted its match horizon: further matching events exist and **no cursor reaches them**, since a semantic page has no `next_cursor`. Never report a truncated page as the full set of matches; narrow the query or add hard filters rather than raising `--limit`. Both surface as a note under the table, and both are absent when nothing applied, so test for `true`. Third, `--sort significance` is a **best-effort snapshot**: an event that changes tier mid-walk can be skipped entirely, so an `--all` walk under that sort is not guaranteed complete and no client-side de-duplication can repair it. When completeness matters, walk the default sort and pass `--significance notable` instead.

    An event's `status` is `active`, `closed`, `merged` or `removed`. The first two are both **live**: `closed` only means the coverage window lapsed, and it can reopen — it is the normal state for anything but the last few days, so do not read it as deleted or as a reason to re-query. Table output shows a Status column only when a tombstone is present, since `active`/`closed` is uniform within a result set and already implied by the dates; read `status` from `--format json` when you need it per row. `nfn events get` likewise omits Status on a live event. In the same way, the events table shows significance as a one-cell `Sig` marker (● notable, ○ routine, · commentary, decoded in the footer) only when the page mixes tiers, and an Articles column only when some event has more than one; a page of a single tier names it in the footer instead. Never parse table output: its columns change with the terminal width and the data.

    Events also reach terminal states that articles and stories do not. Two events found to describe the same development **merge**; an event with no live coverage left, or one found to be incorrect, is **removed**. Both are permanent. Such an event drops out of the normal feed, but its id keeps resolving: `nfn events get <id>` answers with a **tombstone** carrying only `id`, `status`, `merged_into` and `updated_at` — no title, summary, counts or id arrays. The CLI renders exactly those four fields plus a note pointing at the surviving event, so do not read the missing fields as a failed fetch. An `event_id` you cached can become a tombstone at any time.

    Tombstones appear in a list **only** under `--updated-after`, which is therefore the only way to learn that an event you already hold is gone. Poll with your last sync timestamp and `--all`. Two cautions on that walk. Tombstones respect event-grain filters (`--ticker`, `--event-type`, `--significance`, ...) but *not* article-grain ones (`--country`, `--category`, `--sources`, `--source-type`, `--content-type`), having no live articles left to filter by — poll without those for full-fidelity coverage. And the events cursor walk is at-least-once rather than exactly-once, because a merge moves the surviving event earlier in the default order (and an `--updated-after` walk, ordered by update time ascending, re-emits for the same reason). `--all` de-duplicates by `id` for you, **keeping the copy from the latest page** — which is the current one, since the repeat is caused by the record changing. If you follow `--cursor` by hand, de-duplicate the same way round: keep the last copy, not the first. Keeping the first silently discards the tombstone of an event that merges mid-walk, leaving your mirror holding an event the API no longer serves — the exact failure the sync exists to prevent.

19. **Use `--semantic` when the request is thematic, not lexical** — `nfn articles list`, `nfn stories list`, `nfn companies list` and `nfn events list` accept `--semantic`, which matches `--q` by meaning (embedding similarity) instead of keywords. Reach for it when the user describes a theme rather than a term: "defence contract wins and capacity expansion" finds relevant results that share no words with the query. Stick with plain `--q` when they gave you a name, ticker, or exact phrase — lexical search is better at those and costs nothing extra.

    - `--q` is required and must be 2-200 characters.
    - Every other filter still applies as a hard filter, so `--semantic` composes with `--country`, `--market`, `--sector`, date ranges and the rest.
    - Results are a **single page ranked by closeness**. There is no cursor, so `--all`, `--cursor`, and `--updated-after` are rejected; `--sort` is rejected too, because semantic mode defines its own ranking. Use `--limit` to size the page.
    - **A short page does not mean the archive is thin.** A server-side relevance floor drops weak matches, so a semantic page is frequently shorter than `--limit` and can come back empty — a 25-limit query returning 1 row is normal and means "only one thing cleared the bar", not "coverage is missing". Do not report a small semantic result as evidence that little was published; re-run without `--semantic`, or widen the query, before drawing that conclusion. This is separate from `plan_limited` (invariant 15): the floor shortens a page on any plan, so check `plan_limited` to tell the two causes apart.
    - Each result carries `distance`: the raw cosine distance to the query, lower being closer. Articles and stories are ordered by distance blended with a small recency bonus, so equally close matches come out newest-first, while the reported `distance` stays raw. Companies blend distance with a small coverage bonus, so a well-covered company outranks an equally close but barely-covered one. In table mode this appears as a `Distance` column.
    - **What each type matches on:** articles match their summarized text; stories match title, summary and key entities only — the narrative content is *not* part of the match, so a detail that appears only in the body won't be found; companies match name, aliases, country, sector, industry and description; events match their *member articles'* title and summary, and **not** the event's own title or summary, so `--semantic` on events is not a strict improvement on `--q` — an event with an on-topic title whose coverage is oblique matches `--q` and not `--semantic`.
    - **Keep geography out of `--q` on companies.** A place name in the query matches company *names* containing it ("Nordic ...", "Swedish ...") ahead of companies that actually do the thing. Put it in `--market`, `--domicile` or `--country`, which apply as hard filters before the match.
    - Semantic requests come from a **separate budget of 120 per hour per API key**, shared across every resource that accepts `--semantic`. Don't spray semantic queries where one would do.
    - It never silently falls back to keyword matching: if the backend is down the command fails rather than returning lexical results, so a success is always a genuine semantic result.
    - On stories, a plan's story cap clamps the size of the semantic result, so a capped plan gets its closest matches rather than its newest stories. Check `plan_limited` as usual. On events the recent-events cap does *not* apply in semantic mode, so `plan_limited` there reports only that source and age restrictions are in force.
    - On events only, check `data.truncated` as well (invariant 18). **Events' reported `distance` will not look sorted**, and that is correct, not a bug: ranking uses the *mean of an event's two closest* member articles plus a recency bonus, while the `distance` shown is its single closest article's raw value. Do not re-sort by it or treat an out-of-order row as a defect.

### Output Modes

| Goal | Flag | What you get |
|------|------|-------------|
| Human-readable tables | `--format table` (default in TTY) | Tables fitted to the terminal width (`$COLUMNS` when piped), with local short times and a footer giving the count, the `get` command and the next `--cursor` |
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
nfn articles list --fields "id,title,published_at"  # Narrows both the JSON keys and the table columns
```

`--fields` takes **API field names, not the table's column headings**: `first_reported_at`,
not `reported`. An unknown name is rejected locally, and the error lists the names that feed
accepts, so `--fields reported` costs no request. `nfn <command> --help` lists them too.

A narrowed result omits what you did not ask for rather than reporting it empty: the JSON
carries only the fields you named, and the table only their columns. So a key that is present
carries a real value, never a zero one standing in for a field you skipped. Among the fields
you did ask for, one the API has no value for may still be absent rather than empty, so read
absence as "empty or not requested" and let what you passed decide which.

`id` comes back whether or not you ask for it, and keeps its table column. On a `--semantic`
search `distance` also survives, since it is the ranking output rather than a field. Asking for
a field the table cannot show (`article_url`, `summary`) prints a note naming it and pointing at
`--format json`; `nfn event-types --fields id` is all note and no table, because that table has
no id column.

## Quick Reference

| Task | Command |
|------|---------|
| **Articles** | |
| List articles | `nfn articles list --format json` |
| Filter by country | `nfn articles list --country SE,NO` |
| Filter by category | `nfn articles list --category <category-id>` |
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
| Search articles by meaning | `nfn articles list --q "capacity expansion" --semantic` |
| Only listed companies | `nfn articles list --listed` |
| Filter by watchlist | `nfn articles list --watchlist-id <id>` |
| Filter by content type | `nfn articles list --content-type "press release"` |
| Fetch specific IDs | `nfn articles list --ids "id1,id2,id3" --format json` |
| Incremental sync | `nfn articles list --updated-after "2025-03-01T00:00:00Z" --all --format json` |
| Get article details | `nfn articles get <id> --format json` |
| **Stories** | |
| List stories | `nfn stories list --format json` |
| Filter stories by country | `nfn stories list --country SE,NO` |
| Filter stories by category | `nfn stories list --category <category-id>` |
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
| Search stories by text | `nfn stories list --q "battery"` |
| Search stories by meaning | `nfn stories list --q "supply chain disruption" --semantic` |
| Only listed companies | `nfn stories list --listed` |
| Filter by watchlist | `nfn stories list --watchlist-id <id>` |
| Get story details | `nfn stories get <id> --format json` |
| **Events** | |
| List events | `nfn events list --format json` |
| Only material events | `nfn events list --significance notable` |
| See all tiers, grouped | `nfn events list --sort significance` |
| List the event taxonomy | `nfn event-types` |
| Filter by event type | `nfn events list --event-type earnings-report,acquisition-announced` |
| Filter by ticker | `nfn events list --ticker VOLV-B` |
| Filter by company ID | `nfn events list --company <id>` |
| Recently active only | `nfn events list --window 7d` |
| Filter by first-reported range | `nfn events list --first-reported-after 2026-01-01T00:00:00Z` |
| Filter by last-activity range | `nfn events list --last-article-after 2026-03-01T00:00:00Z` |
| Search events by text (feed order, paginates) | `nfn events list --q "battery"` |
| Search events by meaning | `nfn events list --q "defence contract wins" --semantic` |
| Filter by listing market | `nfn events list --market SE` |
| Filter by exchange | `nfn events list --exchange XSTO,XCSE` |
| Only listed companies | `nfn events list --listed` |
| Filter by watchlist | `nfn events list --watchlist-id <id>` |
| Include peripheral company roles | `nfn events list --ticker VOLV-B --include-peripheral` |
| Commentary reacting to an event | `nfn events list --related-to <event-id>` |
| Incremental sync (includes tombstones) | `nfn events list --updated-after "2026-03-01T00:00:00Z" --all --format json` |
| Get event details | `nfn events get <id> --format json` |
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
| Search companies by meaning | `nfn companies list --q "offshore wind installation" --semantic` |
| Filter by exchange | `nfn companies list --exchange XSTO,XCSE` |
| Filter by listing market | `nfn companies list --market SE` |
| Filter by legal domicile | `nfn companies list --domicile SE` |
| Filter by sector | `nfn companies list --sector Financials` |
| Only companies with an active listing | `nfn companies list --listed` |
| Only active companies | `nfn companies list --is-active` |
| Get company details | `nfn companies get VOLV-B --format json` |
| Company articles | `nfn companies articles VOLV-B --limit 5` |
| Primary articles only | `nfn companies articles VOLV-B --primary-only` |
| Company articles, filtered | `nfn companies articles VOLV-B --category <category-id> --published-after 2026-01-01` |
| Company articles, search | `nfn companies articles VOLV-B --q "battery" --sort latest --all` |
| Company stories | `nfn companies stories VOLV-B --limit 5` |
| Company stories, filtered | `nfn companies stories VOLV-B --category <category-id> --published-after 2026-01-01` |
| Company events | `nfn companies events VOLV-B --limit 5` |
| Company events, material only | `nfn companies events VOLV-B --significance notable` |
| Company events, recently active | `nfn companies events VOLV-B --window 30d` |
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
| **Event types** | |
| List event types | `nfn event-types --format json` |
| Filter events by type | `nfn events list --event-type <slug>` |
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
│   ├── Narrow that company's own feed? → nfn companies articles VOLV-B --category <category-id> --published-after 2026-01-01
│   ├── Search within it? → nfn companies articles VOLV-B --q "battery" --sort latest
│   └── Drop passing mentions? → nfn companies articles VOLV-B --primary-only
├── Know the company name but not identifier? → nfn companies list --q "Volvo"
│   └── Multiple matches? → Ask user or note which entity you're using
├── Several companies at once? → nfn articles list --ticker VOLV-B,ERIC-B
├── Know the topic? → nfn search "topic" --type articles
├── Know the exact word or name? → nfn articles list --q "battery"
├── Know the theme but not the words? → nfn articles list --q "capacity expansion" --semantic
├── Want a specific country? → nfn articles list --country SE
├── Want a whole national market (all its venues)? → nfn articles list --market SE
├── Want companies legally domiciled somewhere? → nfn articles list --domicile SE
├── Want a specific source? → nfn sources list (to see IDs)
│   └── then → nfn articles list --sources <id1,id2>
├── Want a kind of source (exchange feeds, wire services...)? → nfn articles list --source-type stock_exchange
│   └── Fastest disclosure tier → nfn articles list --source-type stock_exchange
├── Want a specific category? → nfn categories (to see IDs)
│   └── then → nfn articles list --category <category-id>
├── Want stories (clustered articles)? → nfn stories list
├── Want what happened, not every article about it? → nfn events list --significance notable
│   ├── For one company? → nfn companies events VOLV-B --significance notable
│   ├── A kind of development? → nfn event-types (to see slugs)
│   │   └── then → nfn events list --event-type earnings-report
│   ├── Only recently active? → nfn events list --window 7d
│   ├── Holding an article or story? → read its event_id, then nfn events get <event-id>
│   └── What changed since a sync? → nfn events list --updated-after <ts> --all
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
├── Know what it does but not its name? → nfn companies list --q "offshore wind installation" --semantic
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
nfn companies articles VOLV-B --category <category-id> --published-after 2026-01-01 --format json
# ...or a full-text search of the company's own coverage, newest first, all pages
nfn companies articles VOLV-B --q "battery" --sort latest --all --format json
# Drop articles where the company is only a passing mention
nfn companies articles VOLV-B --primary-only --format json
```

### Summarise what happened, without reading the same story four times

```bash
# One row per development instead of one per article. --significance notable drops
# the commentary and routine filings the unfiltered feed also returns.
nfn events list --significance notable --window 24h --format json

# The detail view carries the member article IDs, the involved companies with their
# association tier, and the earliest primary-disclosure article (an exchange feed,
# wire service, PR platform or government body) — start there, not at a rewrite.
nfn events get <event-id> --format json

# member_figures_disagree flags a development where sources report different
# numbers for the same figure. figure_divergence gives the range and the
# per-article values behind it; quote the primary disclosure, not the spread.
```

### Go from one article to the whole development

```bash
# Any article or story carries the event it belongs to
nfn articles get <article-id> --format json | jq -r '.data.event_id'

# Then pull the event to find the rest of the coverage
nfn events get <event-id> --format json

# A merged or removed event still resolves, to a tombstone: id, status,
# merged_into and updated_at only. Follow merged_into to the survivor.
```

### Track one company's developments

```bash
# The fourth company member route, alongside articles, stories and calendar events
nfn companies events VOLV-B --limit 10 --format json

# Material developments in the last month
nfn companies events VOLV-B --significance notable --window 30d --format json

# The path fixes the company, so --ticker/--company/--listed/--exchange and the
# other company-selecting filters are rejected here. Use the flat feed for those,
# and for searching one company's events by meaning:
nfn events list --company <id> --q "capacity expansion" --semantic --format json
```

### Keep a local mirror of events in sync

```bash
# Poll with your last completed sync timestamp. This is the ONLY request that
# returns merge and removal tombstones, so it is the only way to learn that an
# event you already hold is gone.
nfn events list --updated-after "2026-03-01T00:00:00Z" --all --format json

# Omit the article-grain filters (--country, --category, --sources, --source-type,
# --content-type) while polling: tombstones have no live articles to match them,
# so those filters silently drop the deletions you are polling for.
# Advance your stored timestamp only after --all has drained every page.
```

### Find the right event type slug

```bash
# --event-type on events takes hyphenated Events-taxonomy slugs, NOT the
# underscored calendar types — each rejects the other's values with a 400
nfn event-types --format json

nfn events list --event-type earnings-report,acquisition-announced --format json
```

### Search for news on a topic

```bash
# Cross-resource search
nfn search "battery" --format json

# Or search just articles with text query
nfn articles list --q "battery" --limit 10 --format json
```

### Search by meaning rather than keyword

```bash
# Lexical: matches the word "battery"
nfn articles list --q "battery" --limit 10 --format json

# Semantic: matches the idea, including results that never say these words
nfn articles list --q "defence contract wins and capacity expansion" \
  --semantic --limit 10 --format json

# Filters still apply as hard filters alongside the semantic match
nfn articles list --q "grid capacity constraints" --semantic \
  --market SE --published-after 2026-01-01 --format json

# Find companies by what they do rather than what they are called
nfn companies list --q "offshore wind turbine installation" --semantic --format json

# Results are one page ranked by closeness; read the distances directly
nfn articles list --q "cost cutting programmes" --semantic --format json \
  | jq '.data.articles[] | {title, distance}'
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
# See available categories and their IDs — --category takes the ID, not the name
nfn categories

# Get Swedish articles in a specific category
nfn articles list --country SE --category <category-id> --limit 10
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
