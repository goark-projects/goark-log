module goark.dev/goark-log/benchmarks/compare

go 1.25

require (
	github.com/rs/zerolog v1.35.1
	go.uber.org/zap v1.28.0
	goark.dev/goark-log v0.0.0
)

require (
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace goark.dev/goark-log => ../..
