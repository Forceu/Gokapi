// Package languages provides the translations for the Gokapi web interface.
//
// All translations are stored as flat JSON files in the translations folder and
// are embedded into the binary. English is always used as a fallback: if a
// translation is missing a key, the English string is displayed instead.
//
// Strings may contain the placeholders {0}, {1}, ... which are replaced by the
// arguments passed to Tf / Hf. A few strings intentionally contain HTML markup
// (e.g. links or spans that JavaScript writes into) - those are rendered with
// H / Hf and must keep their tags and ids when being translated.
package languages

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// translationFiles contains all embedded translation files
//
//go:embed translations/*.json
var translationFiles embed.FS

// CookieName is the name of the cookie that stores the language selected by the user
const CookieName = "gokapi_language"

// DefaultCode is the language that is used if no other language matches the request
const DefaultCode = "en"

// keyLanguageName is the key that contains the native name of the language
const keyLanguageName = "language_name"

// Language contains the metadata of an available translation
type Language struct {
	// Code is the ISO 639-1 code of the language, e.g. "de"
	Code string
	// Name is the name of the language in that language itself, e.g. "Deutsch"
	Name string
}

// Translator contains all strings of a single language and is passed to the templates
type Translator struct {
	// Code is the ISO 639-1 code of the language, e.g. "de"
	Code string
	// Name is the name of the language in that language itself, e.g. "Deutsch"
	Name string

	entries  map[string]string
	jsonData template.JS
}

var (
	// translators contains all loaded languages, indexed by their language code
	translators = make(map[string]Translator)
	// available contains all loaded languages, sorted by their code
	available []Language
	// fallback contains the English strings, which are used if a key is missing
	fallback map[string]string
)

func init() {
	files, err := translationFiles.ReadDir("translations")
	if err != nil {
		panic(err)
	}
	rawEntries := make(map[string]map[string]string)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		content, err := translationFiles.ReadFile("translations/" + file.Name())
		if err != nil {
			panic(err)
		}
		entries := make(map[string]string)
		err = json.Unmarshal(content, &entries)
		if err != nil {
			panic("invalid translation file " + file.Name() + ": " + err.Error())
		}
		rawEntries[strings.TrimSuffix(file.Name(), ".json")] = entries
	}

	var ok bool
	fallback, ok = rawEntries[DefaultCode]
	if !ok {
		panic("translation file for default language " + DefaultCode + " is missing")
	}

	for code, entries := range rawEntries {
		// Merging with the English strings, so that an incomplete translation does
		// not result in empty strings being sent to the browser
		merged := make(map[string]string, len(fallback))
		for key, value := range fallback {
			merged[key] = value
		}
		for key, value := range entries {
			if value != "" {
				merged[key] = value
			}
		}
		encoded, err := json.Marshal(merged)
		if err != nil {
			panic(err)
		}
		translators[code] = Translator{
			Code:     code,
			Name:     merged[keyLanguageName],
			entries:  merged,
			jsonData: template.JS(encoded),
		}
		available = append(available, Language{Code: code, Name: merged[keyLanguageName]})
	}
	sort.Slice(available, func(i, j int) bool {
		return available[i].Name < available[j].Name
	})
}

// Get returns the Translator for the given language code. If the language is
// unknown, the default language is returned instead.
func Get(code string) Translator {
	translator, ok := translators[normaliseCode(code)]
	if !ok {
		return Default()
	}
	return translator
}

// Default returns the Translator for the default language
func Default() Translator {
	return translators[DefaultCode]
}

// IsAvailable returns true if a translation exists for the given language code
func IsAvailable(code string) bool {
	_, ok := translators[normaliseCode(code)]
	return ok
}

// GetAvailable returns all languages that Gokapi has been translated to, sorted by name
func GetAvailable() []Language {
	result := make([]Language, len(available))
	copy(result, available)
	return result
}

// FromRequest returns the Translator that fits the request best. The language
// selected by the user (stored in a cookie) has priority, otherwise the
// Accept-Language header is used. If neither matches, the default language is
// returned.
func FromRequest(r *http.Request) Translator {
	if r == nil {
		return Default()
	}
	cookie, err := r.Cookie(CookieName)
	if err == nil {
		if translator, ok := translators[normaliseCode(cookie.Value)]; ok {
			return translator
		}
	}
	if translator, ok := fromAcceptLanguage(r.Header.Get("Accept-Language")); ok {
		return translator
	}
	return Default()
}

// fromAcceptLanguage parses an Accept-Language header and returns the best
// matching Translator. The second return value is false if no language matched.
func fromAcceptLanguage(header string) (Translator, bool) {
	type weightedLanguage struct {
		code   string
		weight float64
	}
	var candidates []weightedLanguage
	for _, entry := range strings.Split(header, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		code := entry
		weight := 1.0
		if index := strings.Index(entry, ";"); index != -1 {
			code = strings.TrimSpace(entry[:index])
			for _, parameter := range strings.Split(entry[index+1:], ";") {
				parameter = strings.TrimSpace(parameter)
				if !strings.HasPrefix(parameter, "q=") {
					continue
				}
				parsed, err := strconv.ParseFloat(strings.TrimPrefix(parameter, "q="), 64)
				if err == nil {
					weight = parsed
				}
			}
		}
		if code == "" || weight <= 0 {
			continue
		}
		candidates = append(candidates, weightedLanguage{code: code, weight: weight})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].weight > candidates[j].weight
	})
	for _, candidate := range candidates {
		if translator, ok := translators[normaliseCode(candidate.code)]; ok {
			return translator, true
		}
	}
	return Default(), false
}

// normaliseCode converts a language tag like "de-DE" to the language code "de"
func normaliseCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if index := strings.IndexAny(code, "-_"); index != -1 {
		code = code[:index]
	}
	return code
}

// T returns the translated string for the given key. If the key does not exist
// in this language, the English string is returned. If it does not exist at all,
// the key itself is returned, so that a missing translation is easy to spot.
func (t Translator) T(key string) string {
	if value, ok := t.entries[key]; ok {
		return value
	}
	if value, ok := fallback[key]; ok {
		return value
	}
	return key
}

// Tf returns the translated string for the given key, with the placeholders
// {0}, {1}, ... replaced by the passed arguments
func (t Translator) Tf(key string, args ...any) string {
	return replacePlaceholders(t.T(key), args)
}

// H returns the translated string for the given key as HTML. This must only be
// used for strings that intentionally contain markup, as the content is not
// escaped.
func (t Translator) H(key string) template.HTML {
	return template.HTML(t.T(key))
}

// Hf returns the translated string for the given key as HTML, with the
// placeholders {0}, {1}, ... replaced by the passed arguments. This must only be
// used for strings that intentionally contain markup, as the content is not
// escaped.
func (t Translator) Hf(key string, args ...any) template.HTML {
	return template.HTML(t.Tf(key, args...))
}

// Js returns all strings of this language as a JSON object, so that they can be
// embedded into a script tag and used by the JavaScript function t()
func (t Translator) Js() template.JS {
	if t.jsonData == "" {
		return Default().jsonData
	}
	return t.jsonData
}

// Available returns all languages that Gokapi has been translated to, so that a
// template can render the language selector
func (t Translator) Available() []Language {
	return GetAvailable()
}

// replacePlaceholders replaces {0}, {1}, ... with the passed arguments
func replacePlaceholders(input string, args []any) string {
	for i, arg := range args {
		input = strings.ReplaceAll(input, "{"+strconv.Itoa(i)+"}", fmt.Sprint(arg))
	}
	return input
}
