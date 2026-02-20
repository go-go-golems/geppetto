package turns

//go:generate go run ../../cmd/gen-turns --schema spec/turns_codegen.yaml --out . --section kinds
//go:generate go run ../../cmd/gen-turns --schema spec/turns_codegen.yaml --out . --section keys
//go:generate go run ../../cmd/gen-turns --schema spec/turns_codegen.yaml --section dts --dts-template spec/turns.d.ts.tmpl --dts-out ../doc/types/turns.d.ts
