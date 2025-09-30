module github.com/arthur-debert/nanostore

go 1.23

require (
	github.com/arthur-debert/nanostore/nanostore/ids v0.0.0-00010101000000-000000000000
	github.com/arthur-debert/nanostore/types v0.0.0-00010101000000-000000000000
	github.com/gofrs/flock v0.12.1
	github.com/google/uuid v1.6.0
	github.com/rs/zerolog v1.31.0
	github.com/spf13/cobra v1.10.1
)

replace github.com/arthur-debert/nanostore/types => ./types

replace github.com/arthur-debert/nanostore/nanostore/ids => ./nanostore/ids

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
