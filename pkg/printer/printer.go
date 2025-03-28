package printer

import (
	"SettingsSentry/logger"
	"fmt"
)

var (
	AppLogger *logger.Logger
)

type Printer struct {
	appName     string
	baseAppName string
	firstPrint  bool
}

func NewPrinter(appName string, logger *logger.Logger) *Printer {
	AppLogger = logger

	p := &Printer{
		baseAppName: appName,
		firstPrint:  true,
	}
	p.SetAppName(appName)
	return p
}

func (p *Printer) SetAppName(appName string) {
	p.baseAppName = appName
	if appName != "" {
		p.appName = "\n\033[1m" + appName + "\033[0m -> "
	} else {
		p.appName = ""
	}
	p.firstPrint = true
}

func (p *Printer) Print(format string, args ...interface{}) {
	if AppLogger == nil {
		fmt.Printf("Logger not initialized for Printer. Message: "+format+"\n", args...)
		return
	}

	message := fmt.Sprintf(format, args...)

	if p.firstPrint && p.appName != "" {
		AppLogger.Logf("%s%s", p.appName, message)
		p.firstPrint = false
	} else {
		if !p.firstPrint && p.appName != "" {
			AppLogger.Logf("  %s", message)
		} else {
			AppLogger.Logf("%s", message)
		}
	}
}

func (p *Printer) Reset() {
	p.firstPrint = true
}
