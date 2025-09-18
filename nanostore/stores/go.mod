module github.com/arthur-debert/nanostore/stores

go 1.21

replace github.com/arthur-debert/nanostore/types => ../../types
replace github.com/arthur-debert/nanostore/ids => ../ids

require (
	github.com/arthur-debert/nanostore/types v0.0.0-00010101000000-000000000000
	github.com/arthur-debert/nanostore/ids v0.0.0-00010101000000-000000000000
	github.com/gofrs/flock v0.8.1
	github.com/google/uuid v1.3.0
)