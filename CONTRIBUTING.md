# Midgard Contributors

Welcome!

## Quick Links:
**API Docs:** https://testnet.midgard.thorchain.info/v2/doc
**Bugs/Issue Tracker:** https://gitlab.com/thorchain/midgard/-/boards
**Discord, ask for help:** https://discord.gg/fQtKn8Xe (THORChain Dev Discord, `#midgard` Channel) 

## How to report a bug
Create a new issue and select the `bug_report` template from the **Type** Dropdown menue.

## How to request a new feature/enhancement

Create a new issue and select the `feature_request` template from the **Type** Dropdown menue.

# Development

## Environment Setup

**Requirements**
1. Docker - https://www.docker.com/get-started
2. Golang - https://golang.org/

* TODO - add more specific instructions

## Format / Linting
You can run these before submit to make sure the CI will pass:

```bash
gofmt -l -s -w ./
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint golangci-lint run -v
```
## Testing

* TODO

## Update Documentation / Inline comments
If you have made changes or added an API endpoint please make sure the documentation reflects your changes. Please, make sure to include the units of a value for example: `// in Rune` or `// in Asset`.

Please comment your code! We all depend on your comments to build together!

## How to submit changes

    1. Create an issue describing the body of work you plan to contribute 
    2. Create a PR with the changes
    3. Use this template: 
    4. Share PR `#midgard` Discord channel

## Review Process
The community will review, if the fix/functionality is benificial it will be slotted to be merged for specific version release.  