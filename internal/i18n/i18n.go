package i18n

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	sml "codeberg.org/crowdware/sml-go"
)

var SupportedLanguages = []string{"de", "en", "eo", "pt", "fr", "es"}

type Bundle struct {
	strings map[string]map[string]string
}

func Load(dir string) (*Bundle, error) {
	bundle := &Bundle{
		strings: make(map[string]map[string]string, len(SupportedLanguages)),
	}

	for _, lang := range SupportedLanguages {
		path := filepath.Join(dir, fmt.Sprintf("strings-%s.sml", lang))
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("warning: missing i18n file: %s", path)
				continue
			}
			return nil, fmt.Errorf("read i18n file %s: %w", path, err)
		}

		doc, err := sml.ParseDocument(string(raw))
		if err != nil {
			return nil, fmt.Errorf("parse i18n file %s: %w", path, err)
		}

		root := findNodeByName(doc.Roots, "Strings")
		if root == nil {
			return nil, fmt.Errorf(`i18n file %s: missing root node "Strings"`, path)
		}

		values := make(map[string]string, len(root.Properties))
		for _, prop := range root.Properties {
			values[prop.Name] = prop.Value
		}
		bundle.strings[lang] = values
	}

	if _, ok := bundle.strings["en"]; !ok {
		log.Printf("warning: missing i18n fallback language: en")
		bundle.strings["en"] = map[string]string{}
	}

	return bundle, nil
}

func (b *Bundle) T(lang, key string) string {
	if b == nil {
		return key
	}

	if langMap, ok := b.strings[lang]; ok {
		if val, found := langMap[key]; found {
			return val
		}
	}

	if enMap, ok := b.strings["en"]; ok {
		if val, found := enMap[key]; found {
			return val
		}
	}

	return key
}

func (b *Bundle) Map(lang string) map[string]string {
	result := make(map[string]string)
	if b == nil {
		return result
	}

	src, ok := b.strings[lang]
	if !ok {
		src = b.strings["en"]
	}

	for k, v := range src {
		result[k] = v
	}

	return result
}

func DetectLang(r *http.Request, supported []string) string {
	if r == nil {
		return "en"
	}

	cookie, err := r.Cookie("lang")
	if err == nil && isSupported(cookie.Value, supported) {
		return cookie.Value
	}

	return "en"
}

func findNodeByName(nodes []sml.Node, name string) *sml.Node {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
	}
	return nil
}

func isSupported(lang string, supported []string) bool {
	for _, item := range supported {
		if lang == item {
			return true
		}
	}
	return false
}
