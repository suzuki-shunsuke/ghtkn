module github.com/suzuki-shunsuke/ghtkn

go 1.26.3

require (
	github.com/google/go-cmp v0.7.0
	github.com/lmittmann/tint v1.1.3
	github.com/spf13/afero v1.15.0
	github.com/suzuki-shunsuke/ghtkn-go-sdk v0.2.2
	github.com/suzuki-shunsuke/slog-error v0.2.2
	github.com/suzuki-shunsuke/slog-util v0.3.2
	github.com/suzuki-shunsuke/urfave-cli-v3-util v0.2.3
	github.com/urfave/cli/v3 v3.9.0
	golang.org/x/crypto v0.52.0
	golang.org/x/term v0.43.0
)

// replace github.com/suzuki-shunsuke/ghtkn-go-sdk v0.2.2 => ../ghtkn-go-sdk
replace github.com/suzuki-shunsuke/ghtkn-go-sdk v0.2.2 => github.com/suzuki-shunsuke/ghtkn-go-sdk v0.2.3-0.20260601103132-400152040231

require (
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/google/go-github/v88 v88.0.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/suzuki-shunsuke/go-error-with-exit-code v1.0.0 // indirect
	github.com/suzuki-shunsuke/go-exec v0.0.1 // indirect
	github.com/zalando/go-keyring v0.2.8 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
