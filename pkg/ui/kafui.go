package ui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// openUIFunc is a variable that holds the OpenUI function, allowing it to be mocked in tests
var openUIFunc = OpenUI

func Init(cfgOption string, useMock bool) {
	// Initialize debug logging (clears old log and sets up rotation)
	shared.InitDebugLog()

	fmt.Println("Init...")
	var dataSource api.KafkaDataSource

	dataSource = mock.KafkaDataSourceMock{}
	if !useMock {
		dataSource = kafds.NewKafkaDataSourceKaf()
	}
	dataSource.Init(cfgOption)
	openUIFunc(dataSource)
}

func OpenUI(dataSource api.KafkaDataSource) {
	p := tea.NewProgram(initialModel(dataSource), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
