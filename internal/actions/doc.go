// Package actions — action catalogue (v1.0)
//
// report           → no filesystem mutation; every file is ActionKeep or
//                    ActionIgnore. See planReport().
// copy-unique      → every unique + canonical becomes ActionCopy with
//                    DstPath = OutputMapping.OutputPath. See planCopyUnique().
// move-duplicates  → every non-canonical duplicate becomes ActionMove; the
//                    canonical is ActionKeep. See planMoveDuplicates().
// delete-duplicates → every non-canonical duplicate becomes ActionDelete. The
//                    CLI layer enforces --yes before calling Execute.
//                    See planDeleteDuplicates().
//
// Every mode produces an ActionPlan sorted by lexical SrcPath and numbered by
// Seq. See Docs/09-design-actions.md for the authoritative description.
package actions
