package kafui

type UIEvent string

const (
	OnModalClose  UIEvent = "ModalClose"
	OnFocusSearch UIEvent = "FocusSearch"
)
