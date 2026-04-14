# Security Policy

## Supported Versions

Only the latest release receives security updates.

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

Please report security issues via [GitHub's private vulnerability reporting](https://github.com/nordic-financial-news/nfn-cli/security/advisories/new).

You should receive an acknowledgement within 5 business days. We will work with you to understand and address the issue before any public disclosure.

## Scope

The following are in scope:

- API key leakage or credential exposure
- Command injection or argument injection
- Dependency vulnerabilities
- Insecure defaults

The following are out of scope:

- Vulnerabilities in the upstream Nordic Financial News API
- Issues requiring physical access to the machine

## Security Model

- **API keys** are stored in your operating system's keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager) — not in plaintext config files.
- **All API communication** uses HTTPS. The CLI rejects non-HTTPS API URLs.
- **No shell execution** is performed by the CLI. User input is URL-encoded, not interpolated into commands.
