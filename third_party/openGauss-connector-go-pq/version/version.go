package main

import (
	"fmt"
)

var (
	version     = "v1.0.2"
	productline = "h0.csi.gaussdb_kernel"
	productname = "opengaussgo"
	versionid   = "r4"
)

func main() {
	fmt.Printf("%s-%s.%s.%s\n", version, productline, productname, versionid)
}
