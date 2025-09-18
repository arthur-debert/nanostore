module samples-todos

go 1.23

require github.com/arthur-debert/nanostore v0.0.0

require (
	github.com/arthur-debert/nanostore/types v0.0.0-00010101000000-000000000000 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
)

replace (
	github.com/arthur-debert/nanostore => ../../
	github.com/arthur-debert/nanostore/search => ../../search
	github.com/arthur-debert/nanostore/types => ../../types
)
