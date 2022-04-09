package main

import (
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
)

func main() {
	myutils.RunTimeInit()
	logger, slogger = myutils.InitLogger("main", myconst.DebugLevel)

	findLoadBalance("Donut")

}


