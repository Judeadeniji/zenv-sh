module github.com/Judeadeniji/zenv-sh/cli

go 1.25.0

require (
	github.com/Judeadeniji/zenv-sh/amnesia v0.0.0
	github.com/spf13/cobra v1.10.2
)

replace github.com/Judeadeniji/zenv-sh/amnesia => ../amnesia

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
