# come

come is a declarative dsl with trollish keywords that compiles into vertically-sliced go apis.

## quickstart

```bash
go build ./cmd/comec
comec init myapi
cd myapi
comec add feature users
comec build
cd generated
go run ./cmd/server
```

## keywords

| keyword | purpose |
|---------|---------|
| `nogo` | app name |
| `pileup` | database (postgres/sqlite) |
| `aura` | server config (port, timeouts) |
| `unblockthehomies` | cors origin |
| `bouncer` | jwt auth config |
| `borrow` | import feature folder |
| `manifest` | data model |
| `pick` | enum type |
| `yeet` | http route |
| `ward` | middleware on route |
| `vouch` | request validation |
| `grabit` | database query |
| `hurl` | response definition |
| `spawnchaos` | seed data |
| `vibes` | environment variables |
| `rawgo` | inline go code |
| `spotlight` | database index |
| `reshape` | custom migration |
| `homie` | foreign key relationship |

## full documentation

see [come.md](come.md) for the complete language reference, project structure, and examples.

## architecture

- **compiler**: zero-dependency go (pure stdlib)
- **generated code**: stdlib `net/http`, raw `database/sql`, vertical slice architecture
- **databases**: postgresql + sqlite
- **auth**: built-in jwt
