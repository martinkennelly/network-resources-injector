package service

type Service interface {
	Run() error
	Quit() error
	StatusSignal() chan struct{}
	GetName() string
}
