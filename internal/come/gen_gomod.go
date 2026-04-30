package come

import "fmt"

func GenGoMod(proj *Project) string {
	return fmt.Sprintf(`module %s

go 1.23.0

require (
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.31.0
	modernc.org/sqlite v1.34.5
)

require (
	github.com/dustin/go-humanize v1.0.1
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/mattn/go-isatty v0.0.20
	github.com/ncruces/go-strftime v0.1.9
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec
	golang.org/x/sys v0.28.0
	modernc.org/libc v1.61.0
	modernc.org/mathutil v1.6.0
	modernc.org/memory v1.8.0
)
`, proj.AppName)
}
