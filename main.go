package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
)

func main() {
	threshold := flag.Int("threshold", 15, "inlining budget to test for")
	all := flag.Bool("all", false, "check all packages, not just current")
	flag.Parse()

	gcflags := "-gcflags="
	if *all {
		gcflags += "all="
	}
	gcflags += "-m -m"
	args := []string{"build", "-a", gcflags}
	args = append(args, flag.Args()...)

	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, string(out))
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type Match struct {
		File   string
		Line   int
		Func   string
		Cost   int
		Budget int
	}

	var found []Match

	rx := regexp.MustCompile(`(?m)^(.*):(\d+):\d+: cannot inline (.*): function too complex: cost (\d+) exceeds budget (\d+)$`)
	for _, matches := range rx.FindAllStringSubmatch(string(out), -1) {
		var fn Match
		var err error

		fn.File = matches[1]
		if fn.Line, err = strconv.Atoi(matches[2]); err != nil {
			continue
		}
		fn.Func = matches[3]
		if fn.Cost, err = strconv.Atoi(matches[4]); err != nil {
			continue
		}

		if fn.Budget, err = strconv.Atoi(matches[5]); err != nil {
			continue
		}

		if fn.Cost-fn.Budget > *threshold {
			continue
		}

		found = append(found, fn)
	}

	sort.Slice(found, func(i, k int) bool {
		return found[i].Cost < found[k].Cost
	})

	for _, fn := range found {
		fmt.Printf("%s:%d: func %s cost +%d\n",
			fn.File, fn.Line, fn.Func, fn.Cost-fn.Budget)
	}
}
