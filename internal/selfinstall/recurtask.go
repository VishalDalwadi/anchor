package selfinstall

// RecurTaskName is the name of the scheduled/periodic job anchor installs
// to run `anchor recur run`.
const RecurTaskName = "AnchorRecurRun"

// SetupRecurTask installs a periodic system trigger that runs
// `anchor recur run` against the binary at anchorPath, so recurring
// templates actually spawn tasks without the user remembering to run it
// themselves. Per anchor-spec.md §4, `recur run` is idempotent and meant
// to be driven by Windows Task Scheduler, not invoked by other commands.
// Returns a human-readable message describing what happened.
func SetupRecurTask(anchorPath string) (string, error) {
	return setupRecurTask(anchorPath)
}
