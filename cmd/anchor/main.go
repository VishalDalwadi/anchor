// Command anchor externalizes working memory: tasks, tiered goals, a
// passive watchlist, and cron-driven recurring tasks.
package main

import (
	"fmt"
	"os"

	"github.com/VishalDalwadi/anchor/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "anchor:", err)
		os.Exit(1)
	}
}
