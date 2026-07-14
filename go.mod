module github.com/suzuki-shunsuke/ghtkn

go 1.26.5

// replace github.com/suzuki-shunsuke/ghtkn-go-sdk v0.4.1 => ../ghtkn-go-sdk

// TEMPORARY: develop against the local SDK checkout for the LoadConfig work.
// Remove this and bump the ghtkn-go-sdk version to the release before merging.
// replace github.com/suzuki-shunsuke/ghtkn-go-sdk => ../ghtkn-go-sdk

// replace github.com/suzuki-shunsuke/go-github-device-flow v0.0.1 => ../go-github-device-flow

require (
	github.com/google/go-cmp v0.7.0
	github.com/lmittmann/tint v1.1.3
	github.com/spf13/afero v1.15.0
	github.com/suzuki-shunsuke/gen-go-jsonschema v0.1.0
	github.com/suzuki-shunsuke/ghtkn-go-sdk v0.4.2-0.20260714223732-e30eb547415a
	github.com/suzuki-shunsuke/go-github-device-flow v0.0.2-0.20260714121453-3389c27e8995
	github.com/suzuki-shunsuke/go-revoke-github-access-token v0.0.1
	github.com/suzuki-shunsuke/slog-error v0.2.2
	github.com/suzuki-shunsuke/slog-util v0.3.2
	github.com/suzuki-shunsuke/urfave-cli-v3-util v0.2.3
	github.com/urfave/cli/v3 v3.10.1
	golang.design/x/clipboard v0.8.0
	golang.org/x/crypto v0.54.0
	golang.org/x/term v0.45.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/invopop/jsonschema v0.12.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/suzuki-shunsuke/go-error-with-exit-code v1.0.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/zalando/go-keyring v0.2.8 // indirect
	golang.design/x/x11 v0.2.0 // indirect
	golang.org/x/exp/shiny v0.0.0-20250606033433-dcc06ee1d476 // indirect
	golang.org/x/image v0.28.0 // indirect
	golang.org/x/mobile v0.0.0-20250606033058-a2a15c67f36f // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)
