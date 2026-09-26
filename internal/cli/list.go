package cli

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VishalDalwadi/anchor/internal/idgen"
	"github.com/VishalDalwadi/anchor/internal/model"
)

// Collections ("lists") are deliberately simple: a name, freeform text
// items, and an optional parent list, with none of a task's due dates,
// aspects or status. Lists are referred to by name (or id); names are
// unique across all lists, so a sublist can be named directly without
// spelling out its parents.

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Keep simple named lists (movies to watch, ...), optionally nested as sublists",
	}
	cmd.AddCommand(newListCreateCmd())
	cmd.AddCommand(newListAddCmd())
	cmd.AddCommand(newListRemoveCmd())
	cmd.AddCommand(newListShowCmd())
	cmd.AddCommand(newListAllCmd())
	cmd.AddCommand(newListEditCmd())
	cmd.AddCommand(newListDeleteCmd())
	return cmd
}

func newListCreateCmd() *cobra.Command {
	var parent string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a list (a sublist with --parent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCollections(func(cols []model.Collection) ([]model.Collection, error) {
				name := strings.TrimSpace(args[0])
				if err := validateCollectionName(cols, name, ""); err != nil {
					return nil, err
				}
				c := model.Collection{ID: idgen.New("c"), Name: name, Items: []string{}}
				if parent != "" {
					p, err := findCollection(cols, parent)
					if err != nil {
						return nil, err
					}
					c.Parent = cols[p].ID
				}
				fmt.Fprintln(cmd.OutOrStdout(), c.ID)
				return append(cols, c), nil
			})
		},
	}
	cmd.Flags().StringVar(&parent, "parent", "", "list to create this inside of, as a sublist")
	registerCollectionFlag(cmd, "parent")
	return cmd
}

func newListAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "add <list> <item>",
		Short:             "Add an item to a list",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeCollectionArg(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCollections(func(cols []model.Collection) ([]model.Collection, error) {
				i, err := findCollection(cols, args[0])
				if err != nil {
					return nil, err
				}
				item := strings.TrimSpace(args[1])
				if item == "" {
					return nil, fmt.Errorf("item text is empty")
				}
				cols[i].Items = append(cols[i].Items, item)
				return cols, nil
			})
		},
	}
}

func newListRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <list> <item-number-or-text>",
		Short: "Remove an item from a list, by its number in `list show` or its text",
		Long: `Remove an item from a list, by its number as shown by ` + "`list show`" + `, or by
its text (exact, or ignoring case). A number that's out of range is tried as
text, so an item like "1917" can still be removed by name.`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeCollectionArg(true),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCollections(func(cols []model.Collection) ([]model.Collection, error) {
				i, err := findCollection(cols, args[0])
				if err != nil {
					return nil, err
				}
				j, err := findItem(cols[i].Items, args[1])
				if err != nil {
					return nil, fmt.Errorf("list %s: %w", cols[i].Name, err)
				}
				cols[i].Items = append(cols[i].Items[:j], cols[i].Items[j+1:]...)
				return cols, nil
			})
		},
	}
}

func newListShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "show <list>",
		Short:             "Show a list's items, and its sublists' items nested beneath",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeCollectionArg(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			cols, err := loadCollections()
			if err != nil {
				return err
			}
			i, err := findCollection(cols, args[0])
			if err != nil {
				return err
			}
			printCollection(cmd.OutOrStdout(), cols, cols[i], 0)
			return nil
		},
	}
}

func newListAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Show every list and sublist, with item counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cols, err := loadCollections()
			if err != nil {
				return err
			}
			if len(cols) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no lists yet (anchor list create <name>)")
				return nil
			}
			printCollectionTree(cmd.OutOrStdout(), cols)
			return nil
		},
	}
}

func newListEditCmd() *cobra.Command {
	var name, parent string
	cmd := &cobra.Command{
		Use:               "edit <list>",
		Short:             "Rename a list, or move it under another list (--parent=, to make it top-level)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeCollectionArg(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCollections(func(cols []model.Collection) ([]model.Collection, error) {
				i, err := findCollection(cols, args[0])
				if err != nil {
					return nil, err
				}
				if cmd.Flags().Changed("parent") {
					newParent := ""
					if parent != "" {
						p, err := findCollection(cols, parent)
						if err != nil {
							return nil, err
						}
						if err := validateCollectionParent(cols, cols[i].ID, cols[p].ID); err != nil {
							return nil, err
						}
						newParent = cols[p].ID
					}
					cols[i].Parent = newParent
				}
				if cmd.Flags().Changed("name") {
					n := strings.TrimSpace(name)
					if err := validateCollectionName(cols, n, cols[i].ID); err != nil {
						return nil, err
					}
					cols[i].Name = n
				}
				return cols, nil
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new name")
	cmd.Flags().StringVar(&parent, "parent", "", `list to move this under ("--parent=" to make it top-level)`)
	registerCollectionFlag(cmd, "parent")
	return cmd
}

func newListDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:               "delete <list>",
		Short:             "Delete a list (one with items needs --force; one with sublists must be emptied of them first)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeCollectionArg(false),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCollections(func(cols []model.Collection) ([]model.Collection, error) {
				i, err := findCollection(cols, args[0])
				if err != nil {
					return nil, err
				}
				c := cols[i]
				if subs := sublists(cols, c.ID); len(subs) > 0 {
					var names []string
					for _, s := range subs {
						names = append(names, s.Name)
					}
					return nil, fmt.Errorf("list %s has sublists (%s); delete them, or move them out with `list edit <sublist> --parent=`, first",
						c.Name, strings.Join(names, ", "))
				}
				if len(c.Items) > 0 && !force {
					return nil, fmt.Errorf("list %s has %d item(s); use --force to delete it anyway", c.Name, len(c.Items))
				}
				return append(cols[:i], cols[i+1:]...), nil
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "delete even if the list still has items")
	return cmd
}

func loadCollections() ([]model.Collection, error) {
	s, err := openStore()
	if err != nil {
		return nil, err
	}
	return s.LoadCollections()
}

// withCollections loads all lists, applies change, and saves the result
// unless change fails.
func withCollections(change func([]model.Collection) ([]model.Collection, error)) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	cols, err := s.LoadCollections()
	if err != nil {
		return err
	}
	cols, err = change(cols)
	if err != nil {
		return err
	}
	return s.SaveCollections(cols)
}

// findCollection looks a list up by id, else by name (ignoring case).
func findCollection(cols []model.Collection, nameOrID string) (int, error) {
	for i, c := range cols {
		if c.ID == nameOrID {
			return i, nil
		}
	}
	for i, c := range cols {
		if strings.EqualFold(c.Name, nameOrID) {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no list named %q (see `anchor list all`)", nameOrID)
}

// validateCollectionName checks name is usable for the list selfID
// (empty for a new list): non-empty, and not taken by another list.
func validateCollectionName(cols []model.Collection, name, selfID string) error {
	if name == "" {
		return fmt.Errorf("list name is empty")
	}
	for _, c := range cols {
		if c.ID != selfID && strings.EqualFold(c.Name, name) {
			return fmt.Errorf("a list named %q already exists", c.Name)
		}
	}
	return nil
}

// validateCollectionParent refuses to put list childID under parentID if
// that's the list itself or one of its own sublists (a loop).
func validateCollectionParent(cols []model.Collection, childID, parentID string) error {
	byID := map[string]model.Collection{}
	for _, c := range cols {
		byID[c.ID] = c
	}
	for id := parentID; id != ""; id = byID[id].Parent {
		if id == childID {
			return fmt.Errorf("list %s can't be moved under %s: that's itself or one of its own sublists",
				byID[childID].Name, byID[parentID].Name)
		}
		if _, ok := byID[id]; !ok {
			break
		}
	}
	return nil
}

// findItem resolves an item reference: a 1-based number within range,
// else the item's exact text, else its text ignoring case.
func findItem(items []string, ref string) (int, error) {
	if n, err := strconv.Atoi(ref); err == nil && n >= 1 && n <= len(items) {
		return n - 1, nil
	}
	for i, it := range items {
		if it == ref {
			return i, nil
		}
	}
	for i, it := range items {
		if strings.EqualFold(it, ref) {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no item %q (use its number from `list show`, or its text)", ref)
}

// sublists returns the lists directly inside id, sorted by name.
func sublists(cols []model.Collection, id string) []model.Collection {
	var subs []model.Collection
	for _, c := range cols {
		if c.Parent == id && c.ID != id {
			subs = append(subs, c)
		}
	}
	sort.Slice(subs, func(i, j int) bool { return strings.ToLower(subs[i].Name) < strings.ToLower(subs[j].Name) })
	return subs
}

// printCollection prints c's name, its numbered items, then each sublist
// the same way, indented beneath it.
func printCollection(w io.Writer, cols []model.Collection, c model.Collection, depth int) {
	printCollectionAt(w, cols, c, depth, map[string]bool{})
}

func printCollectionAt(w io.Writer, cols []model.Collection, c model.Collection, depth int, seen map[string]bool) {
	if seen[c.ID] {
		return // a parent loop in hand-edited data
	}
	seen[c.ID] = true
	indent := strings.Repeat("  ", depth)
	name := c.Name
	if depth > 0 {
		name += "/"
	}
	fmt.Fprintln(w, indent+name)
	subs := sublists(cols, c.ID)
	if len(c.Items) == 0 && len(subs) == 0 {
		fmt.Fprintln(w, indent+"  (empty)")
	}
	for i, it := range c.Items {
		fmt.Fprintf(w, "%s  %d. %s\n", indent, i+1, it)
	}
	for _, s := range subs {
		printCollectionAt(w, cols, s, depth+1, seen)
	}
}

// printCollectionTree prints every list as a tree with item counts.
// Lists whose parent is missing are shown at the top level.
func printCollectionTree(w io.Writer, cols []model.Collection) {
	exists := map[string]bool{}
	for _, c := range cols {
		exists[c.ID] = true
	}
	var roots []model.Collection
	for _, c := range cols {
		if c.Parent == "" || !exists[c.Parent] || c.Parent == c.ID {
			roots = append(roots, c)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return strings.ToLower(roots[i].Name) < strings.ToLower(roots[j].Name) })

	seen := map[string]bool{}
	var walk func(c model.Collection, depth int)
	walk = func(c model.Collection, depth int) {
		if seen[c.ID] {
			return
		}
		seen[c.ID] = true
		fmt.Fprintf(w, "%s%s\t(%s)\n", strings.Repeat("  ", depth), c.Name, itemCount(len(c.Items)))
		for _, s := range sublists(cols, c.ID) {
			walk(s, depth+1)
		}
	}
	for _, c := range roots {
		walk(c, 0)
	}
	for _, c := range cols { // anything stuck in a parent loop
		walk(c, 0)
	}
}

func itemCount(n int) string {
	if n == 1 {
		return "1 item"
	}
	return fmt.Sprintf("%d items", n)
}

// completeCollectionArg completes a command's <list> argument with list
// names; with items set, the argument after it with that list's item
// numbers (text shown alongside). Other positions are free text.
func completeCollectionArg(items bool) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch {
		case len(args) == 0:
			return collectionNameCompletions()
		case len(args) == 1 && items:
			s, err := openStore()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			cols, err := s.LoadCollections()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			i, err := findCollection(cols, args[0])
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var out []string
			for j, it := range cols[i].Items {
				out = append(out, idCompletion(strconv.Itoa(j+1), it))
			}
			return out, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func collectionNameCompletions() ([]string, cobra.ShellCompDirective) {
	s, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	cols, err := s.LoadCollections()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var out []string
	for _, c := range cols {
		out = append(out, idCompletion(c.Name, itemCount(len(c.Items))))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// registerCollectionFlag completes a flag that takes a list name.
func registerCollectionFlag(cmd *cobra.Command, flag string) {
	_ = cmd.RegisterFlagCompletionFunc(flag, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return collectionNameCompletions()
	})
}
