package plugin

type Plugin interface {
	Name() string
	Trigger(contextData string)
}
