package main

// ZACA: multilingual (ES/FR) support for subscriber-facing pieces.
//
// listmonk upstream is single-language per instance: one *i18n.I18n built at
// boot from `app.lang` serves admin, public pages and system e-mails alike.
// This file adds a small, self-contained store that lazily builds and caches a
// per-language *i18n.I18n (reusing upstream's getI18nLang) plus a per-language
// set of notification e-mail templates, so subscriber-facing public pages and
// the double opt-in e-mail can be rendered in the subscriber's own language
// while the admin stays on the instance default.
//
// The store is intentionally decoupled from *App: the opt-in notify hook is
// built (main.go) before *App exists, and both it and the public handlers must
// share the same cache. Nothing here touches the DB schema; the language is
// read from the free-form subscriber.attribs.lang JSON field.
//
// See ZACA-CHANGES.md for the full list of touched files.

import (
	"html/template"
	"strings"
	"sync"

	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/notifs"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/stuffbin"
)

// zacaI18nKey is the echo.Context key under which subscriber-facing handlers
// stash the resolved per-request *i18n.I18n for tplRenderer.Render to pick up.
const zacaI18nKey = "zaca_i18n"

// i18nStore lazily builds and caches per-language i18n instances and per-language
// notification e-mail template sets. Unknown/empty languages fall back to the
// instance default (app.lang), preserving upstream behaviour.
type i18nStore struct {
	fs  stuffbin.FileSystem
	u   *UrlConfig
	def *i18n.I18n // the instance default (app.lang), used as fallback.

	mu    sync.RWMutex
	langs map[string]*i18n.I18n
	tpls  map[string]*template.Template
}

// newI18nStore builds a store. `def` is the boot-time default i18n instance.
func newI18nStore(fs stuffbin.FileSystem, u *UrlConfig, def *i18n.I18n) *i18nStore {
	return &i18nStore{
		fs:    fs,
		u:     u,
		def:   def,
		langs: map[string]*i18n.I18n{},
		tpls:  map[string]*template.Template{},
	}
}

// For returns the i18n instance for the given language code, falling back to the
// instance default for empty/unknown codes. It reuses upstream's getI18nLang
// (English base + language overlay).
func (s *i18nStore) For(lang string) *i18n.I18n {
	lang = normLang(lang)
	if lang == "" {
		return s.def
	}

	s.mu.RLock()
	if i, ok := s.langs[lang]; ok {
		s.mu.RUnlock()
		return i
	}
	s.mu.RUnlock()

	// getI18nLang loads English as a base and overlays the selected language.
	i, ok, err := getI18nLang(lang, s.fs)
	if err != nil {
		// ok == false means even the English base couldn't be read; ok == true
		// but err != nil means the selected language file was missing/partial
		// and `i` is plain English. In both cases we prefer the instance
		// default (app.lang) over surprising the subscriber with English.
		_ = ok
		i = s.def
	}

	s.mu.Lock()
	s.langs[lang] = i
	s.mu.Unlock()
	return i
}

// NotifTpls returns the notification e-mail template set bound to the given
// language's i18n (so the funcmap `L` inside the templates resolves in that
// language). Empty/unknown languages fall back to the globally parsed set
// (notifs.Tpls), which is bound to app.lang.
func (s *i18nStore) NotifTpls(lang string) *template.Template {
	lang = normLang(lang)
	if lang == "" || s.For(lang) == s.def {
		return notifs.Tpls
	}

	s.mu.RLock()
	if t, ok := s.tpls[lang]; ok {
		s.mu.RUnlock()
		return t
	}
	s.mu.RUnlock()

	// Parse the e-mail templates with the funcmap bound to this language,
	// mirroring initNotifs().
	t, err := stuffbin.ParseTemplatesGlob(initTplFuncs(s.For(lang), s.u), s.fs, "/static/email-templates/*.html")
	if err != nil {
		// On any parse error, degrade gracefully to the default set.
		return notifs.Tpls
	}

	s.mu.Lock()
	s.tpls[lang] = t
	s.mu.Unlock()
	return t
}

// normLang normalizes a language code: lowercases and strips any region/script
// suffix (es-ES -> es, fr_FR -> fr). Returns "" for empty input.
func normLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return ""
	}
	if i := strings.IndexAny(lang, "-_"); i > 0 {
		lang = lang[:i]
	}
	return lang
}

// subLang extracts the subscriber's language from the free-form attribs.lang
// JSON field. Returns "" when unset or not a string.
func subLang(sub models.Subscriber) string {
	if sub.Attribs == nil {
		return ""
	}
	if v, ok := sub.Attribs["lang"].(string); ok {
		return normLang(v)
	}
	return ""
}

// listsLang derives a language from a set of lists via a `lang:xx` tag. Used as a
// fallback where no subscriber record is available (e.g. the opt-in page). It
// returns the first matching tag found, or "".
func listsLang(lists []models.List) string {
	for _, l := range lists {
		for _, t := range l.Tags {
			if s, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(t)), "lang:"); ok {
				if s = normLang(s); s != "" {
					return s
				}
			}
		}
	}
	return ""
}
