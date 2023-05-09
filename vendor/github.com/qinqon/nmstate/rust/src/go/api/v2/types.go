//go:generate cargo run --manifest-path ../apigen/Cargo.toml -- zz_generated.types.go
//go:generate ./controller-gen.sh object:headerFile="boilerplate.go.txt" paths="."
//go:generate go fmt types.go

package nmstate
