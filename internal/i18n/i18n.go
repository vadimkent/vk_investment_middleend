package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

var (
	translations map[string]map[string]string
	mu           sync.RWMutex
	defaultLang  = "en"
)

// Load reads all locale JSON files from the given directory.
func Load(localesDir string) error {
	mu.Lock()
	defer mu.Unlock()

	translations = make(map[string]map[string]string)

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		lang := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(localesDir, entry.Name()))
		if err != nil {
			log.Warn().Str("file", entry.Name()).Err(err).Msg("failed to read locale file")
			continue
		}

		var nested map[string]any
		if err := json.Unmarshal(data, &nested); err != nil {
			log.Warn().Str("file", entry.Name()).Err(err).Msg("failed to parse locale file")
			continue
		}

		flat := make(map[string]string)
		flatten("", nested, flat)
		translations[lang] = flat

		log.Info().Str("lang", lang).Int("keys", len(flat)).Msg("loaded locale")
	}

	return nil
}

// T translates a key for the given language. Falls back to default language, then to the key itself.
func T(lang, key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if msgs, ok := translations[lang]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}

	if msgs, ok := translations[defaultLang]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}

	return key
}

// flatten converts nested JSON into dot-separated keys.
func flatten(prefix string, m map[string]any, out map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case string:
			out[key] = val
		case map[string]any:
			flatten(key, val, out)
		}
	}
}
