package platformutils

import (
	_ "embed"

	"github.com/playwright-community/playwright-go"
)

//go:embed stealth.min.js
var stealthScript string

// InjectStealthScript 注入 stealth.min.js 隐藏自动化特征
func InjectStealthScript(context playwright.BrowserContext) error {
	err := context.AddInitScript(playwright.Script{
		Content: playwright.String(stealthScript),
	})
	return err
}
