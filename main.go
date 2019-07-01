package main

import (
    "log"

    "github.com/clarkema/sshcat/cmd"
)

func main() {
    // Disable the timestamp prefix on Go's logger
    log.SetFlags(0)

    root.Execute()
}
