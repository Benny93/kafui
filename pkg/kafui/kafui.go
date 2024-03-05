package kafui

import (
	"com/emptystate/kafui/pkg/datasource/mock"
	"fmt"
)

func Init() {

	fmt.Printf("Init...")
	kafMock := mock.KafkaDataSourceMock{}
	OpenUI(kafMock)
}
