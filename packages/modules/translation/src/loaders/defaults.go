// Package loaders mirrors @medusajs/translation/src/loaders.
package loaders

import (
	"context"
	"fmt"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/models"
)

// defaultLocales mirrors defaultLocales (BCP 47 language tags).
var defaultLocales = [][2]string{
	{"en-US", "English (United States)"},
	{"en-GB", "English (United Kingdom)"},
	{"en-AU", "English (Australia)"},
	{"en-CA", "English (Canada)"},
	{"es-ES", "Spanish (Spain)"},
	{"es-MX", "Spanish (Mexico)"},
	{"es-AR", "Spanish (Argentina)"},
	{"fr-FR", "French (France)"},
	{"fr-CA", "French (Canada)"},
	{"fr-BE", "French (Belgium)"},
	{"de-DE", "German (Germany)"},
	{"de-AT", "German (Austria)"},
	{"de-CH", "German (Switzerland)"},
	{"it-IT", "Italian (Italy)"},
	{"pt-BR", "Portuguese (Brazil)"},
	{"pt-PT", "Portuguese (Portugal)"},
	{"nl-NL", "Dutch (Netherlands)"},
	{"nl-BE", "Dutch (Belgium)"},
	{"da-DK", "Danish (Denmark)"},
	{"sv-SE", "Swedish (Sweden)"},
	{"nb-NO", "Norwegian Bokmål (Norway)"},
	{"fi-FI", "Finnish (Finland)"},
	{"pl-PL", "Polish (Poland)"},
	{"lt-LT", "Lithuanian (Lithuania)"},
	{"cs-CZ", "Czech (Czech Republic)"},
	{"sk-SK", "Slovak (Slovakia)"},
	{"hu-HU", "Hungarian (Hungary)"},
	{"ro-RO", "Romanian (Romania)"},
	{"bg-BG", "Bulgarian (Bulgaria)"},
	{"el-GR", "Greek (Greece)"},
	{"tr-TR", "Turkish (Turkey)"},
	{"ru-RU", "Russian (Russia)"},
	{"uk-UA", "Ukrainian (Ukraine)"},
	{"ar-SA", "Arabic (Saudi Arabia)"},
	{"ar-AE", "Arabic (United Arab Emirates)"},
	{"ar-EG", "Arabic (Egypt)"},
	{"he-IL", "Hebrew (Israel)"},
	{"fa-IR", "Persian (Iran)"},
	{"hi-IN", "Hindi (India)"},
	{"bn-BD", "Bengali (Bangladesh)"},
	{"th-TH", "Thai (Thailand)"},
	{"vi-VN", "Vietnamese (Vietnam)"},
	{"id-ID", "Indonesian (Indonesia)"},
	{"ms-MY", "Malay (Malaysia)"},
	{"tl-PH", "Tagalog (Philippines)"},
	{"zh-CN", "Chinese Simplified (China)"},
	{"zh-TW", "Chinese Traditional (Taiwan)"},
	{"zh-HK", "Chinese Traditional (Hong Kong)"},
	{"ja-JP", "Japanese (Japan)"},
	{"ko-KR", "Korean (South Korea)"},
	{"ka-GE", "Georgian (Georgia)"},
	{"mn-MN", "Mongolian (Mongolia)"},
}

// Defaults mirrors defaults.ts: the default locales are upserted (existing
// codes keep their ids).
func Defaults(ctx context.Context, opts modulessdk.LoaderOptions) error {
	svc := utilsmodulessdk.NewMedusaService(opts.DB, models.All, nil).Internal("Locale")
	existing, err := svc.List(ctx, nil, utilsmodulessdk.FindConfig{Select: []string{"id", "code"}})
	if err != nil {
		opts.Logger.Warn(fmt.Sprintf("Failed to load locales, skipping loader. Original error: %s", err))
		return nil
	}
	byCode := map[string]any{}
	for _, l := range existing {
		code, _ := l.Get("code")
		id, _ := l.Get("id")
		byCode[fmt.Sprint(code)] = id
	}
	data := make([]*dal.Record, 0, len(defaultLocales))
	for _, l := range defaultLocales {
		r := dal.NewRecord()
		r.Set("code", l[0])
		r.Set("name", l[1])
		if id, ok := byCode[common.NormalizeLocale(l[0])]; ok && id != nil {
			r.Set("id", id)
		}
		data = append(data, r)
	}
	resp, err := svc.Upsert(ctx, data)
	if err != nil {
		opts.Logger.Warn(fmt.Sprintf("Failed to load locales, skipping loader. Original error: %s", err))
		return nil
	}
	opts.Logger.Debug(fmt.Sprintf("Loaded %d locales", len(resp)))
	return nil
}
