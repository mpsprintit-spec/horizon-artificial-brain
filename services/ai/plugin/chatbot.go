package plugin

import "fmt"

type ChatbotPlugin struct{}

func (p *ChatbotPlugin) Name() string { return "LogSystem" }
func (p *ChatbotPlugin) Trigger(data string) {
	fmt.Printf("   💬 [PLUGINS - CHATBOT] Layar Monitor: 'Aksi %s sukses diproses'\n", data)
}
