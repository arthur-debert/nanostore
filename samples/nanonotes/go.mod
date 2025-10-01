module samples-nanonotes

go 1.23

require (
	github.com/arthur-debert/nanostore v0.0.0
	github.com/spf13/cobra v1.10.1
)

require (
	github.com/arthur-debert/nanostore/nanostore/ids v0.0.0 // indirect
	github.com/arthur-debert/nanostore/types v0.0.0 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/arthur-debert/nanostore => ../../

replace github.com/arthur-debert/nanostore/types => ../../types

replace github.com/arthur-debert/nanostore/nanostore/ids => ../../nanostore/ids
