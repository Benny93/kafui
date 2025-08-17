# Migration Plan: Kafui from tview to Bubble Tea

**Objective:** To systematically migrate the `kafui` TUI from the `rivo/tview` library to the `charmbracelet/bubbletea` framework.

**Guiding Principles:**
- **Atomicity:** Each step is the smallest possible logical change.
- **Verifiability:** Every step concludes with an automated check to ensure the project remains in a working state.
- **Safety:** Each step includes a fallback procedure to revert the change if verification fails.
- **Explicitness:** All commands, file paths, and code changes are stated precisely, requiring no inference.

**Target Executor:** This plan is designed for an autonomous AI agent.

---

### **Step 0: Establish a Healthy Baseline**

**Description:** Before making any changes, we must confirm that the project is currently in a buildable and testable state.

**1. Command to Run:**
```bash
go build . && go test ./...
```

**2. Verification:**
- The `go build` command must complete without any error messages.
- The `go test` command must run and report `ok` for all tested packages. There should be no `FAIL` messages.
- The command must exit with status code 0.

**3. Fallback:**
- If this step fails, **DO NOT PROCEED**. The project is in a broken state that must be fixed manually before attempting migration.

---

### **Step 1: Add Bubble Tea and Supporting Charm Dependencies**

**Description:** Introduce the new dependencies into the project's `go.mod` file. We will add `bubbletea` and its common companion libraries for UI components.

**1. Command to Run:**
```bash
go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest github.com/charmbracelet/lipgloss@latest
```

**2. Verification:**
- The command should execute without errors.
- Run `cat go.mod` and verify that the following lines (or newer versions) have been added:
```
github.com/charmbracelet/bubbles v...
github.com/charmbracelet/bubbletea v...
github.com/charmbracelet/lipgloss v...
```
- Run `go mod tidy` to ensure the module file is clean.

**3. Fallback:**
- If the `go get` command fails, check for network issues.
- If `go.mod` is malformed, revert the changes: `git checkout -- go.mod go.sum`.

---

### **Step 2: Create the Core Bubble Tea Application Shell**

**Description:** We will replace the `tview` application entry point in `main.go` with the basic structure of a `bubbletea` application. This will temporarily break the UI but establish the new foundation.

**1. File to Modify:** `main.go`

**2. Code Change:**
Replace the entire content of `main.go` with this new Bubble Tea entry point.

**Old `main.go` (for context, do not use):**
```go
// ... (original content of main.go, which calls tview)
package main

import (
	"github.com/BenBB/kafui/cmd/kafui"
)

func main() {
	kfui.Execute()
}
```

**New `main.go`:**
```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// mainModel will be the core model for our application.
// For now, it's just a placeholder.
type mainModel struct {
	// state will eventually hold our pages, tables, etc.
}

// Init is the first function that will be called.
func (m mainModel) Init() tea.Cmd {
	return nil // No initial command
}

// Update is called when a message is received.
func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Is it a key press?
	case tea.KeyMsg:
		// Cool, what was the key press?
		switch msg.String() {
		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	// Return the updated model to the Bubble Tea runtime.
	return m, nil
}

// View renders the UI.
func (m mainModel) View() string {
	// For now, just show a message.
	return "Kafui Migration to Bubble Tea in Progress...\n\nPress 'q' to quit."
}


func main() {
	// kafui.Execute() // We will replace this
	p := tea.NewProgram(mainModel{})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
```

**3. Command to Run for Verification:**
```bash
go build .
```

**4. Verification:**
- The `go build` command must complete successfully, creating a `kafui.exe` (or `kafui`) executable.
- Run the executable (`./kafui` or `kafui.exe`). The terminal should clear and display:
  ```
  Kafui Migration to Bubble Tea in Progress...

  Press 'q' to quit.
  ```
- Pressing 'q' or 'Ctrl+C' should exit the application.

**5. Fallback:**
- If the build fails, it's likely due to a copy-paste error. Re-apply the changes carefully.
- If the failure is unrecoverable, revert the file: `git checkout -- main.go`.

---

### **Step 3: Decommission `cmd/kafui/root.go`**

**Description:** The file `cmd/kafui/root.go` contains the `tview` application setup. Since we have a new entry point in `main.go`, this file is now obsolete and must be cleaned to remove `tview` imports.

**1. File to Modify:** `cmd/kafui/root.go`

**2. Code Change:**
Replace the entire content of `cmd/kafui/root.go` with a placeholder function to avoid breaking `go test` which may reference the package.

**New `cmd/kafui/root.go`:**
```go
package kafui

// This file is a placeholder during the tview -> bubbletea migration.
// The original Execute() function set up the tview application, which is now
// handled by the bubbletea program in the main package.

// Execute is a placeholder function.
func Execute() {
	// No-op
}
```

**3. Command to Run for Verification:**
```bash
go build . && go test ./...
```

**4. Verification:**
- Both `go build` and `go test` must pass. Some tests may now be trivial if they tested the old `Execute` function, which is acceptable at this stage. The key is that no compilation or test framework errors occur.

**5. Fallback:**
- If tests fail due to dependencies on the old `Execute` logic, revert the file: `git checkout -- cmd/kafui/root.go`. The tests will need to be refactored in a later step before this file can be decommissioned.

---

### **Step 4: Isolate and Remove `tview` from `pkg/kafui/ui.go`**

**Description:** This is a major step. We will gut the `pkg/kafui/ui.go` file, removing all `tview` code. We will replace it with a new `UI` struct that will manage the Bubble Tea program.

**1. File to Modify:** `pkg/kafui/ui.go`

**2. Code Change:**
This file is complex. The goal is to remove all `tview` imports and logic. Replace its entire content.

**New `pkg/kafui/ui.go`:**
```go
package kafui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// UI is the main struct for the application's user interface.
// It will hold the bubbletea program.
type UI struct {
	program *tea.Program
	// We will add models for pages here later.
}

// NewUI creates a new UI. The logic is simplified for now.
func NewUI(clusterName string) (*UI, error) {
	// The mainModel will be the root of our Bubble Tea application.
	// We will expand this model significantly.
	mainModel := struct{}{} // Placeholder for the root model

	p := tea.NewProgram(mainModel)

	return &UI{
		program: p,
	},
}

// Run starts the UI.
func (ui *UI) Run() error {
	// For now, we create a new placeholder program here.
	// This will be connected to the main.go entry point later.
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// initialModel defines the root model for our application.
// This will eventually replace the temporary models in main.go and here.
type model struct {
	// In the future, this will hold:
	// - table model for topics
	// - text input for search
	// - viewport for details
	// - current page/view state
	message string
}

func initialModel() model {
	return model{
		message: "This is the new Kafui UI powered by Bubble Tea!",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.message + "\n\n(Press q to quit)"
}

// Stop cleanly shuts down the bubbletea program.
func (ui *UI) Stop() {
	if ui.program != nil {
		ui.program.Quit()
	}
}
```

**3. Command to Run for Verification:**
```bash
go test ./...
```

**4. Verification:**
- This command is expected to **FAIL**. The output will list all the files in `pkg/kafui` that depended on the old `ui.go` structure. This is the desired outcome, as it gives us a precise to-do list.
- **Example Expected Failure Output:**
  ```
  # github.com/BenBB/kafui/pkg/kafui_test
  ./kafui_test.go:12:3: undefined: NewUI
  ./page_main_test.go:25:4: undefined: newMainPage
  ...
  ```

**5. Fallback:**
- This step is designed to fail. The "fallback" is to proceed to the next steps, which will fix the errors identified here one by one. If you need to revert, use `git checkout -- pkg/kafui/ui.go`.

---

### **Step 5 through N: Iterative Component Migration**

**Description:** This is a repeating loop where we migrate one `tview` component (or page) to a `bubbletea` model at a time. We will start with `page_main.go`. The process for each file is:
1.  Create a `bubbletea` model for the component.
2.  Give it `Init`, `Update`, and `View` methods.
3.  Remove the old `tview` code.
4.  Fix the tests for that component.

**Example: Migrating `page_main.go`**

**1. File to Modify:** `pkg/kafui/page_main.go`

**2. Code Change:**
Replace the `tview.Table` logic with a `bubbles/table` model.

**New `pkg/kafui/page_main.go` (conceptual):**
```go
package kafui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// mainPageModel represents the main view of the application with the topics table.
type mainPageModel struct {
	table table.Model
}

func newMainPage() mainPageModel {
	columns := []table.Column{
		{Title: "Topic", Width: 40},
		{Title: "Partitions", Width: 15},
		{Title: "Replicas", Width: 15},
	}

	// Dummy data for now

rows := []table.Row{
		{"my-cool-topic", "3", "1"},
		{"another-topic", "12", "3"},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Style the table
	s := table.DefaultStyles()
	ss.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")),
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")),
		Background(lipgloss.Color("57")),
		Bold(false)
	t.SetStyles(s)


	return mainPageModel{table: t}
}

func (m mainPageModel) Init() tea.Cmd {
	return nil
}

func (m mainPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m mainPageModel) View() string {
	return lipgloss.NewStyle().Margin(1).Render(m.table.View())
}
```

**3. Fix Tests:**
- Modify `pkg/kafui/page_main_test.go` to test the new `mainPageModel`. Instead of simulating key presses on a `tview` app, you will create a model, call its `Update` method with a `tea.KeyMsg`, and assert the state of the resulting model.

**4. Verification Loop:**
- After refactoring `page_main.go` and its test, run `go test ./...`.
- The number of failing files should decrease.
- Repeat this process for `page_detail.go`, `page_topic.go`, `search_bar.go`, etc., until `go test ./...` passes completely. Each component will be mapped:
    - `tview.Table` -> `bubbles/table.Model`
    - `tview.TextView` -> `bubbles/viewport.Model`
    - `tview.InputField` -> `bubbles/textinput.Model`
    - `tview.Flex`/`Grid` -> `lipgloss.JoinVertical`/`JoinHorizontal`

**5. Fallback:**
- If a component migration proves too complex, revert the changes for that specific file (e.g., `git checkout -- pkg/kafui/page_main.go pkg/kafui/page_main_test.go`) and try a smaller component first.

---

### **Step N+1: Final Cleanup and Dependency Removal**

**Description:** Once all components are migrated and all tests are passing, we can remove the `tview` dependency entirely.

**1. Command to Run:**
```bash
go mod tidy
```

**2. Verification:**
- Run `cat go.mod`. The line `github.com/rivo/tview` must be gone.
- Run `grep -r "github.com/rivo/tview" .` inside the project root. The command should produce no output, confirming no `.go` files import `tview`.

**3. Fallback:**
- If `tview` is still present in `go.mod`, it means an import was missed. Run `go mod why github.com/rivo/tview` to find the offending package and return to the component migration steps to fix it.

---

### **Step N+2: Final Validation**

**Description:** Run a final, comprehensive script to ensure the migration was a success.

**1. Validation Script:**
Create a script named `validate_migration.sh` with the following content:

```bash
#!/bin/bash
set -e

echo "--- [1/5] Building final binary ---"
go build .
if [ $? -ne 0 ]; then
    echo "ERROR: Final build failed."
    exit 1
fi
echo "Build successful."

echo "--- [2/5] Running all tests ---"
go test ./...
if [ $? -ne 0 ]; then
    echo "ERROR: Tests failed after migration."
    exit 1
fi
echo "All tests passed."

echo "--- [3/5] Checking for leftover tview code imports ---"
TVIEW_CODE_IMPORTS=$(grep -r "github.com/rivo/tview" ./**/*.go || true)
if [ -n "$TVIEW_CODE_IMPORTS" ]; then
    echo "ERROR: Found leftover tview imports in code:"
    echo "$TVIEW_CODE_IMPORTS"
    exit 1
fi
echo "No tview code imports found."

echo "--- [4/5] Checking for tview in go.mod ---"
TVIEW_MOD_ENTRY=$(grep "tview" go.mod || true)
if [ -n "$TVIEW_MOD_ENTRY" ]; then
    echo "ERROR: tview is still present in go.mod:"
    echo "$TVIEW_MOD_ENTRY"
    exit 1
fi
echo "tview successfully removed from go.mod."

echo "--- [5/5] Launching application for manual smoke test ---"
echo "The application will start now. Please manually verify its core functionality."
echo "Press Ctrl+C to exit the smoke test."
./kafui # or kafui.exe on Windows

echo ""
echo "-------------------------------------------"
echo "✅ MIGRATION VALIDATION COMPLETE AND PASSED"
echo "-------------------------------------------"
```

**2. Command to Run:**
```bash
chmod +x validate_migration.sh
./validate_migration.sh
```

**3. Verification:**
- The script must run to completion and print the final "MIGRATION VALIDATION COMPLETE AND PASSED" message.
- The application should launch, be visually correct, and respond to basic interactions.

**4. Fallback:**
- If any step in the validation script fails, it will print an error and exit. Address the specific error reported (e.g., re-run `go mod tidy`, find the leftover import). The migration is not complete until this script passes.
