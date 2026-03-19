module github.com/Judeadeniji/zenv-sh/wasm

go 1.24

require github.com/Judeadeniji/zenv-sh/amnesia v0.0.0

require (
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
)

replace github.com/Judeadeniji/zenv-sh/amnesia => ../amnesia
