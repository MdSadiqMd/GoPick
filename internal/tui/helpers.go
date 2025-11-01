package tui

func (m Model) ShouldPrintCommands() bool {
	return m.quitWithCommands
}

func (m Model) GetCommandsToPrint() []string {
	return m.commandsToPrint
}

func (m Model) ShouldAutoRun() bool {
	return m.autoRun
}
