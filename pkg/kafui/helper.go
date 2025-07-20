package kafui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

// https://stackoverflow.com/a/70802740
func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

// https://stackoverflow.com/questions/37562873/most-idiomatic-way-to-select-elements-from-an-array-in-golang
func filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

// Implement the sort.Interface
type ByOffsetThenPartition []api.Message

func (a ByOffsetThenPartition) Len() int      { return len(a) }
func (a ByOffsetThenPartition) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByOffsetThenPartition) Less(i, j int) bool {
	// First, compare by Offset
	if a[i].Offset != a[j].Offset {
		return a[i].Offset < a[j].Offset
	}
	// If Offset values are equal, then compare by Partition
	return a[i].Partition < a[j].Partition
}

func RecoverAndExit(app *tview.Application) {
	if r := recover(); r != nil {
		app.Stop()
		fmt.Println("An error occurred:", r)
		//debug.PrintStack()
		fmt.Println("Application stopped.")
	}
}
