# AI Agent Prompts for Bubble Tea Migration

## Initial Start Prompt
```
You are tasked with migrating the Kafui project from tview to Bubble Tea UI framework. Follow the migration plan in bubbleteamigration.md exactly. For each step:

1. Read and understand the current step's requirements
2. Execute the commands exactly as written
3. Verify the results match the expected output
4. If verification passes, proceed to next step
5. If verification fails, execute the fallback procedure


After each step, respond with:
- Step completed: [step number and name]
- Actions taken: [list of commands/changes made]
- Verification results: [output of verification steps]
- Next step: [next step to be executed]
- Current status: [success/partial/failed]

write progress to progress.md (append only style)
Before proceeding, confirm you have read and understood the migration plan in bubbleteamigration.md.
```

## Resume Prompt
```
You are continuing the Kafui project migration from tview to Bubble Tea UI framework. To resume:

1. First, assess the current state:
   - Check current git branch and status
   - Locate the last completed step in bubbleteamigration.md and progress.md

2. Continue with the migration plan from the appropriate point

Before proceeding, confirm you have assessed the current state and identified the correct next step.
```

## Error Recovery Prompt
```
An error has occurred during the Kafui migration process. To recover:

1. Analyze the error:
   - Error message: [paste error message]
   - Step where error occurred: [step number and name]
   - Expected vs actual outcome: [describe difference]

2. Check if this is a known failure case:
   - Review fallback procedures in bubbleteamigration.md
   - Check if error matches any predicted failure scenarios

3. Execute recovery:
   - If known failure: Follow documented fallback procedure
   - If unknown failure:
     a. Save current work state
     b. Analyze error root cause
     c. Propose and execute minimal recovery steps
     d. Verify system returns to known good state

4. Before resuming:
   - Verify all tests pass in current state
   - Confirm repository is in clean state
   - Document new failure case and solution

Proceed with recovery steps and report results before continuing with migration.
```

## Verification Prompt
```
Verify the current state of the Kafui migration:

1. Execute these checks:
   ```powershell
   # Build check
   go build ./...
   
   # Test check
   go test ./...
   
   # Import check
   go list -f '{{.ImportPath}} {{.Imports}}' ./... | findstr "tview"
   
   # Git status
   git status
   ```

2. Report results:
   - Build status: [success/fail]
   - Test status: [pass/fail]
   - Remaining tview imports: [yes/no]
   - Git state: [clean/dirty]

3. Analyze results:
   - If all checks pass: Continue with next step
   - If any check fails: Execute appropriate fallback procedure

Provide full results and recommended next actions.
```

## Final Validation Prompt
```
Perform final validation of the Kafui Bubble Tea migration:

1. Execute validation script:
   ```powershell
   ./scripts/validate_migration.sh
   ```

2. Perform manual checks:
   - [ ] Application starts without errors
   - [ ] All UI components render correctly
   - [ ] Navigation works as expected
   - [ ] Search functionality works
   - [ ] Topic management functions work
   - [ ] No visual artifacts or layout issues
   - [ ] Performance is acceptable

3. Document results:
   - Automated validation: [pass/fail]
   - Manual checks: [list results]
   - Performance metrics: [if applicable]
   - Any unexpected behavior: [describe]

4. Final status:
   - [ ] All validations passed
   - [ ] Documentation updated
   - [ ] Ready for PR
   - [ ] Additional fixes needed [list if any]

Provide comprehensive validation report and recommendation for project status.
```

## Progress Tracking Format
For consistent tracking across sessions, use this format in commit messages:

```
bubbletea-migration: Step X.Y - Brief description

- Completed: [list of completed tasks]
- Verified: [list of verification steps passed]
- Next: [next step number]

[Optional: Any issues or notes for next session]
```

This format helps resume work by clearly indicating progress and next steps.
