package main

import (
    "log"

    "git.sr.ht/~clarkema/sshcat/cmd"
)

func main() {
    // Disable the timestamp prefix on Go's logger
    log.SetFlags(0)

    root.Execute()
}
