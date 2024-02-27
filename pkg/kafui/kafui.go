package kafui

import "fmt"

func Init() {

	fmt.Printf("Init...")
	kafMock := KafkaDataSourceMock{}
	OpenUI(kafMock)
}
