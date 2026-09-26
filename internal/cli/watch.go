package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/aspect"
	"github.com/VishalDalwadi/anchor/internal/idgen"
	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/validate"
)

// staleAfterDays is how long a watch item can go unchecked before it's
// considered stale by `watch list --stale`.
const staleAfterDays = 14

func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Manage the passive watchlist",
	}
	cmd.AddCommand(newWatchAddCmd())
	cmd.AddCommand(newWatchEditCmd())
	cmd.AddCommand(newWatchCheckCmd())
	cmd.AddCommand(newWatchResolveCmd())
	cmd.AddCommand(newWatchListCmd())
	return cmd
}

func newWatchAddCmd() *cobra.Command {
	var aspectFlag, expected, notes string
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a watchlist item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aspect.Validate(aspectFlag); err != nil {
				return err
			}
			expectedDate, err := validate.FlexDate(expected, time.Now())
			if err != nil {
				return err
			}
			if notes, err = readNotes(cmd, notes, ""); err != nil {
				return err
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			items, err := s.LoadWatchItems()
			if err != nil {
				return err
			}

			w := model.WatchItem{
				ID:          idgen.New("w"),
				Text:        args[0],
				Aspect:      aspectFlag,
				Expected:    expectedDate,
				LastChecked: validate.Today(),
				Status:      model.WatchWaiting,
				Notes:       notes,
			}
			items = append(items, w)
			if err := s.SaveWatchItems(items); err != nil {
				return err
			}
			fmt.Println(w.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "life aspect (required)")
	cmd.Flags().StringVar(&expected, "expected", "", dateUsage("expected resolution date"))
	cmd.Flags().StringVar(&notes, "notes", "", notesUsage("notes"))
	cmd.MarkFlagRequired("aspect")
	registerAspectFlag(cmd)
	return cmd
}

func newWatchEditCmd() *cobra.Command {
	var text, expected, notes string
	cmd := &cobra.Command{
		Use:               "edit <id>",
		Short:             "Edit a watchlist item",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeWatchIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if cmd.Flags().Changed("expected") {
				if expected, err = validate.FlexDate(expected, time.Now()); err != nil {
					return err
				}
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			items, err := s.LoadWatchItems()
			if err != nil {
				return err
			}
			idx := findWatch(items, args[0])
			if idx < 0 {
				return fmt.Errorf("no watchlist item with id %q", args[0])
			}
			if notes, err = readNotes(cmd, notes, items[idx].Notes); err != nil {
				return err
			}
			if cmd.Flags().Changed("text") {
				items[idx].Text = text
			}
			if cmd.Flags().Changed("expected") {
				items[idx].Expected = expected
			}
			if cmd.Flags().Changed("notes") {
				items[idx].Notes = notes
			}
			return s.SaveWatchItems(items)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "new text")
	cmd.Flags().StringVar(&expected, "expected", "", dateUsage("new expected date"))
	cmd.Flags().StringVar(&notes, "notes", "", notesUsage("new notes"))
	return cmd
}

func newWatchCheckCmd() *cobra.Command {
	var notes string
	cmd := &cobra.Command{
		Use:               "check <id>",
		Short:             "Record a check-in on a watchlist item",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeWatchIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			items, err := s.LoadWatchItems()
			if err != nil {
				return err
			}
			idx := findWatch(items, args[0])
			if idx < 0 {
				return fmt.Errorf("no watchlist item with id %q", args[0])
			}
			if notes, err = readNotes(cmd, notes, items[idx].Notes); err != nil {
				return err
			}
			items[idx].LastChecked = validate.Today()
			if cmd.Flags().Changed("notes") {
				items[idx].Notes = notes
			}
			if items[idx].Status == model.WatchStalled {
				items[idx].Status = model.WatchWaiting
			}
			return s.SaveWatchItems(items)
		},
	}
	cmd.Flags().StringVar(&notes, "notes", "", notesUsage("notes for this check-in"))
	return cmd
}

func newWatchResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "resolve <id>",
		Short:             "Mark a watchlist item resolved",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeWatchIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			items, err := s.LoadWatchItems()
			if err != nil {
				return err
			}
			idx := findWatch(items, args[0])
			if idx < 0 {
				return fmt.Errorf("no watchlist item with id %q", args[0])
			}
			items[idx].Status = model.WatchResolved
			return s.SaveWatchItems(items)
		},
	}
}

func newWatchListCmd() *cobra.Command {
	var stale bool
	var aspectFlag string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List watchlist items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if aspectFlag != "" {
				if err := aspect.Validate(aspectFlag); err != nil {
					return err
				}
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			items, err := s.LoadWatchItems()
			if err != nil {
				return err
			}

			for _, w := range items {
				if w.Status == model.WatchResolved {
					continue
				}
				if aspectFlag != "" && w.Aspect != aspectFlag {
					continue
				}
				if stale && !isStale(w) {
					continue
				}
				printWatch(w)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&stale, "stale", false, fmt.Sprintf("only items unchecked for %d+ days", staleAfterDays))
	cmd.Flags().StringVar(&aspectFlag, "aspect", "", "filter by life aspect")
	registerAspectFlag(cmd)
	return cmd
}

func isStale(w model.WatchItem) bool {
	if w.Status == model.WatchStalled {
		return true
	}
	last, err := time.Parse(validate.DateLayout, w.LastChecked)
	if err != nil {
		return false
	}
	return int(time.Since(last).Hours()/24) >= staleAfterDays
}

func printWatch(w model.WatchItem) {
	line := fmt.Sprintf("%s\t[%s]\t%-8s\t%s\t(last checked %s)", w.ID, w.Status, w.Aspect, w.Text, w.LastChecked)
	if w.Expected != "" {
		line += "\t(expected " + w.Expected + ")"
	}
	fmt.Println(line)
}

func findWatch(items []model.WatchItem, id string) int {
	for i, w := range items {
		if w.ID == id {
			return i
		}
	}
	return -1
}
