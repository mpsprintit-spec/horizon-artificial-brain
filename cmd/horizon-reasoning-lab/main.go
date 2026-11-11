// horizon-reasoning-lab runs an isolated numeric cognition experiment.
// It does not initialize or modify the production Horizon brain.
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"

    nf "github.com/project-horizon/horizon-core/internal/nativefoundation"
)

func main() {
    out:=flag.String("out","","new output directory (must not already exist)")
    seed:=flag.Int64("seed",73,"experiment seed")
    blocks:=flag.Int("blocks",192,"matched factual training pairs")
    cases:=flag.Int("cases",96,"independent held-out test cases per split")
    flag.Parse()
    if *out=="" {
        fmt.Fprintln(os.Stderr,"-out requires a fresh directory")
        os.Exit(2)
    }
    result,err:=nf.RunReasoningLab(*out,*seed,*blocks,*cases)
    if err!=nil {
        fmt.Fprintln(os.Stderr,err)
        os.Exit(1)
    }
    data,err:=json.MarshalIndent(result,"","  ")
    if err!=nil {
        fmt.Fprintln(os.Stderr,err)
        os.Exit(1)
    }
    fmt.Println(string(data))
}
