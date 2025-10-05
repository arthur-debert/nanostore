module github.com/arthur-debert/nanostore

go 1.24.0

require (
	github.com/arthur-debert/nanostore/nanostore/ids v0.0.0-00010101000000-000000000000
	github.com/arthur-debert/nanostore/types v0.0.0-00010101000000-000000000000
	github.com/gofrs/flock v0.12.1
	github.com/google/uuid v1.6.0
	github.com/spf13/cobra v1.10.1
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/arthur-debert/nanostore/types => ./types

replace github.com/arthur-debert/nanostore/nanostore/ids => ./nanostore/ids

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
