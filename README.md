# nfn

Nordic business news from your terminal.

`nfn` searches and filters Swedish, Danish, Finnish, Norwegian and Icelandic business reporting, machine-translated into English. One command answers "what happened at Volvo this week" across 70+ sources publishing in several languages.

An agent can run that command as readily as you can. Every response carries a JSON envelope naming the commands that follow from it, and `nfn commands` hands over the whole catalog as structured data. The agent works out what the tool does on its own, instead of you pasting documentation into a prompt.

- **Self-describing.** `nfn commands` returns every command with its flags, arguments and examples as JSON. The repo ships a [skill file](skills/nfn-cli/SKILL.md) encoding the rules that are not obvious from the help text.
- **One event per development.** Every outlet covering a development files its own story: one Skanska contract drew 14 articles in 42 minutes. `nfn events` returns the development as one row, with a canonical title, a significance tier and the articles attached.
- **Nordic-wide.** Denmark, Finland, Iceland, Norway and Sweden, plus the Faroe Islands, Greenland and the Åland Islands. 70+ sources, 14 exchanges, 115+ event types.
- **Semantic search.** `--semantic` matches on embeddings, so "defence contract wins and capacity expansion" finds results that share no words with the query.

```bash
nfn articles list --limit 3
# osrnn0t30h9x  Iceye and Nokia have entered a new market…   FI  Yle
# byspmxoo0f6u  Hörður steps down as CEO of Landsvirkjun     IS  Vísir
# unm85ku7slpy  Hörður leaves Landsvirkjun                   IS  Viðskiptablaðið

# Those last two are one development, reported twice. Events collapse it:
nfn events get wx5ay6jca6h6
# Title         Landsvirkjun CEO Hörður Arnarson to Leave, Interim CEO From Nov 1
# Type          CEO Departure (ceo-departure)
# Significance  notable
# …
# Articles      2
```

---

**Contents:** [Getting started](#getting-started) · [AI agents](#ai-agents) · [Commands](#commands) · [Output](#output) · [License](#license)

## Getting started

**1. Install.**

```bash
brew install nordic-financial-news/tap/nfn
```

Or, with a Go toolchain:

```bash
go install github.com/nordic-financial-news/nfn-cli/cmd/nfn@latest
```

Or download a binary for your platform from the [releases page](https://github.com/nordic-financial-news/nfn-cli/releases).

**2. Get an API key** from [nordicfinancialnews.com/settings/api_keys](https://nordicfinancialnews.com/settings/api_keys).

**3. Authenticate.** This stores the key in your system keyring, with masked input:

```bash
nfn auth login
```

In CI, or anywhere a keyring is unavailable, set an environment variable instead:

```bash
export NFN_API_KEY=your-api-key
```

**4. Run your first query.**

```bash
nfn articles list --limit 5
```

**5. Check everything is working.** This verifies config, connectivity, authentication and version:

```bash
nfn doctor
```

### Everyday queries

```bash
# Today's notable developments, not routine filings or opinion pieces
nfn events list --significance notable --window 24h

# One company, by ticker
nfn companies events VOLV-B --limit 5

# Swedish and Norwegian articles only
nfn articles list --country SE,NO

# Thematic rather than lexical
nfn events list --q "battery factory investment" --semantic

# Pipe-friendly: JSON is automatic when stdout is not a terminal
nfn articles list --limit 10 | jq -r '.data.articles[].title'
```

> Tickers are written as `VOLV-B`, never with an exchange suffix like `VOLV-B.ST`.

## AI agents

`nfn` works with any agent that can run shell commands. Install the bundled skill:

```bash
npx skills add https://github.com/nordic-financial-news/nfn-cli/skills/nfn-cli
```

Two things make the CLI self-describing, so an agent does not need the docs in its context window:

- **`nfn commands`** returns the full catalog as JSON: every command, flag, argument, example and related command. Narrow it with `nfn commands --command "events list"`.
- **Breadcrumbs.** Every JSON response carries a `breadcrumbs` array suggesting the next command. An agent can walk from a company to its events to the articles behind them without guessing at the data model.

## Commands

| Command | Description |
|---------|-------------|
| `nfn articles list` | List articles (filters for country, category, ticker, etc.) |
| `nfn articles get <id>` | Get article details |
| `nfn stories list` | List stories |
| `nfn stories get <id>` | Get story details |
| `nfn events list` | List events (the articles reporting one development, grouped) |
| `nfn events get <id>` | Get event details |
| `nfn calendar list` | List calendar events (earnings, dividends, AGMs, etc.) |
| `nfn calendar get <id>` | Get calendar event details |
| `nfn companies list` | List companies |
| `nfn companies get <identifier>` | Get company details (by ID or ticker) |
| `nfn companies articles <identifier>` | List articles for a company |
| `nfn companies stories <identifier>` | List stories for a company |
| `nfn companies events <identifier>` | List events for a company |
| `nfn exchanges list` | List exchanges |
| `nfn exchanges get <mic>` | Get exchange details |
| `nfn exchanges companies <mic>` | List companies on an exchange |
| `nfn indices list` | List stock indices |
| `nfn indices get <symbol>` | Get index details |
| `nfn indices companies <symbol>` | List companies in an index |
| `nfn categories` | List article categories |
| `nfn event-types` | List the event taxonomy (values for `--event-type` on events) |
| `nfn countries list` | List supported countries |
| `nfn countries get <code>` | Get country details |
| `nfn sources list` | List news sources (filter articles/stories with `--sources`) |
| `nfn watchlists list` | List your watchlists |
| `nfn watchlists get <id>` | Get watchlist details |
| `nfn search <query>` | Search across articles, stories, and companies |
| `nfn auth login` | Authenticate with your API key |
| `nfn auth status` | Show authentication status and rate limits |
| `nfn auth logout` | Remove stored API key |
| `nfn doctor` | Check CLI health and configuration |
| `nfn commands` | Print the full command catalog as JSON |
| `nfn completion <shell>` | Generate a shell completion script |
| `nfn version` | Print CLI version |

Use `nfn <command> --help` for all available flags.

## Output

- **Interactive terminal:** table format.
- **Piped or scripted:** JSON format.
- Override either with `--format table` or `--format json`.

Tables fit the terminal: the title column takes the spare width and is cut with `…`, and on a narrow terminal the least useful columns drop out first. The ID column is never cut, since every `get` command takes it. Piped tables follow `$COLUMNS`, falling back to 120. Times are shown in your local zone, as short as they can be (`09:42` today, `2 Oct 17:10` this year). JSON always carries the full ISO 8601 values. A footer under each list gives the count, the command that opens a row and, when there is another page, the `--cursor` to reach it.

JSON responses use an envelope, `{ok, data, summary, breadcrumbs}`. The payload is always at `.data`, and errors are always `{ok: false, error}`.

**Two flags mark an incomplete result:**

- `data.plan_limited: true` means your plan hid records from the result. A short list is not evidence that few records exist.
- `data.truncated: true` means a semantic search exhausted its match horizon: further matches exist and no cursor reaches them. Narrow the query rather than raising `--limit`.

Both are omitted entirely when they do not apply, so test for `true`. In table output each surfaces as a note under the table.

Cursor pagination is automatic with `--all`, which follows every page and de-duplicates as it goes.

## License

MIT
