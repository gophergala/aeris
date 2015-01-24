package main

import (
    "flag"
    "fmt"

    "github.com/gophergala/aeris/info"
)

func main() {

    flag.Parse()

    id := flag.Arg(0)

    fmt.Println("fetching info for " + id)

    videoInfo := info.NewInfo(id)

    err := videoInfo.Fetch()
    if err != nil {
        panic(err)
    }

}
