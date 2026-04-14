# nfn-cli — Nordic Financial News CLI

A command-line client for the [Nordic Financial News API](https://nordicfinancialnews.com).

## Install

```bash
brew install nordic-financial-news/tap/nfn
```

Or download a binary from the [releases page](https://github.com/nordic-financial-news/nfn-cli/releases).

## Quick start

```bash
# Authenticate with your API key
nfn auth login

# List recent articles
nfn articles list --limit 5

# Search for companies
nfn search "Volvo"

# Get company details
nfn companies get VOLV-B

# Filter articles by country and category
nfn articles list --country SE --category "Earnings & Financial Results"

# Pipe-friendly — outputs JSON automatically when not in a terminal
nfn articles list --limit 10 | jq '.data.articles[].title'

# Or use an environment variable instead of the keyring
export NFN_API_KEY=your-api-key
nfn articles list --limit 5
```

## Commands

| Command | Description |
|---------|-------------|
| `nfn articles list` | List articles (with filters for country, category, ticker, etc.) |
| `nfn articles get <id>` | Get article details |
| `nfn stories list` | List stories |
| `nfn stories get <id>` | Get story details |
| `nfn companies list` | List companies |
| `nfn companies get <identifier>` | Get company details (by ID or ticker) |
| `nfn companies articles <identifier>` | List articles for a company |
| `nfn companies stories <identifier>` | List stories for a company |
| `nfn exchanges list` | List exchanges |
| `nfn exchanges get <mic>` | Get exchange details |
| `nfn exchanges companies <mic>` | List companies on an exchange |
| `nfn indices list` | List stock indices |
| `nfn indices get <symbol>` | Get index details |
| `nfn indices companies <symbol>` | List companies in an index |
| `nfn categories` | List article categories |
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
| `nfn version` | Print CLI version |

Use `nfn <command> --help` for all available flags.

## Output

- **Interactive terminal (TTY):** table format
- **Piped/scripted:** JSON format
- Override with `--format table` or `--format json`

## AI Agent Integration

`nfn` works with any AI agent that can run shell commands.
Point your agent at [`skills/nfn-cli/SKILL.md`](skills/nfn-cli/SKILL.md) for Nordic financial news workflow coverage.

```bash
npx skills add https://github.com/nordic-financial-news/nfn-cli/skills/nfn-cli
```

**Agent discovery:** Use `nfn commands` for the full command catalog as structured JSON (flags, arguments, examples, related commands).

## License

MIT
