// fixrepo.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	hardcodedIdentifier = "com.ripestore.source"
	hardcodedSourceURL  = "https://raw.githubusercontent.com/RipeStore/repos/main/RipeStore_feather.json"
)

// Root has fields in the order we want them to appear in output JSON.
type Root struct {
	Name         string     `json:"name,omitempty"`
	Subtitle     string     `json:"subtitle,omitempty"`
	Identifier   string     `json:"identifier,omitempty"`
	SourceURL    string     `json:"sourceURL,omitempty"`
	Description  string     `json:"description,omitempty"`
	IconURL      string     `json:"iconURL,omitempty"`
	Website      string     `json:"website,omitempty"`
	PatreonURL   string     `json:"patreonURL,omitempty"`
	HeaderURL    string     `json:"headerURL,omitempty"`
	TintColor    string     `json:"tintColor,omitempty"`
	FeaturedApps []string   `json:"featuredApps,omitempty"`
	Apps         []App      `json:"apps,omitempty"`
	News         []NewsItem `json:"news,omitempty"`
}

type App struct {
	Name                 string          `json:"name,omitempty"`
	BundleIdentifier     string          `json:"bundleIdentifier,omitempty"`
	DeveloperName        string          `json:"developerName,omitempty"`
	Subtitle             string          `json:"subtitle,omitempty"`
	LocalizedDescription string          `json:"localizedDescription,omitempty"`
	IconURL              string          `json:"iconURL,omitempty"`
	TintColor            string          `json:"tintColor,omitempty"`
	Category             string          `json:"category,omitempty"`
	ScreenshotURLs       []string        `json:"screenshotURLs,omitempty"`
	Versions             []Version       `json:"versions,omitempty"`
	AppPermissions       json.RawMessage `json:"appPermissions,omitempty"`
	// marketplaceID, patreon and buildVersion intentionally omitted
}

type Version struct {
	Version              string `json:"version,omitempty"`
	Date                 string `json:"date,omitempty"`
	LocalizedDescription string `json:"localizedDescription,omitempty"`
	DownloadURL          string `json:"downloadURL,omitempty"`
	Size                 int64  `json:"size,omitempty"`
	MinOSVersion         string `json:"minOSVersion,omitempty"`
	// buildVersion intentionally removed
}

type NewsItem struct {
	Title      string      `json:"title,omitempty"`
	Identifier string      `json:"identifier,omitempty"`
	Caption    string      `json:"caption,omitempty"`
	Date       string      `json:"date,omitempty"`
	TintColor  string      `json:"tintColor,omitempty"`
	ImageURL   string      `json:"imageURL,omitempty"`
	Notify     bool        `json:"notify,omitempty"`
	URL        string      `json:"url,omitempty"`
	AppID      interface{} `json:"appID,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run fixrepo.go input.json")
		os.Exit(1)
	}
	inPath := os.Args[1]
	b, err := ioutil.ReadFile(inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read error:", err)
		os.Exit(2)
	}

	// quick UTF-8 sanity: if file bytes not valid UTF-8 we still attempt to recover
	if !utf8.Valid(b) {
		// convert to string and re-decode runes, which replaces invalid sequences with RuneError
		b = []byte(replaceInvalidUTF8(string(b)))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		fmt.Fprintln(os.Stderr, "json parse:", err)
		os.Exit(3)
	}

	out := Root{}

	getStr := func(m map[string]interface{}, k string) string {
		if v, ok := m[k]; ok {
			// treat explicit null as empty string
			if v == nil {
				return ""
			}
			switch vv := v.(type) {
			case string:
				return sanitizeString(vv)
			case float64:
				// number -> string
				return fmt.Sprintf("%v", vv)
			case bool:
				return fmt.Sprintf("%v", vv)
			default:
				// if someone put an object where a string was expected, try to marshal it to string
				if marsh, err := json.Marshal(vv); err == nil {
					return sanitizeString(string(marsh))
				}
			}
		}
		return ""
	}

	out.Name = getStr(raw, "name")
	out.Subtitle = getStr(raw, "subtitle")
	out.Identifier = defaultIfEmpty(getStr(raw, "identifier"), hardcodedIdentifier)
	out.SourceURL = defaultIfEmpty(getStr(raw, "sourceURL"), hardcodedSourceURL)
	out.Description = getStr(raw, "description")
	out.IconURL = getStr(raw, "iconURL")
	out.Website = getStr(raw, "website")
	out.PatreonURL = getStr(raw, "patreonURL")
	out.HeaderURL = getStr(raw, "headerURL")
	out.TintColor = getStr(raw, "tintColor")

	// featuredApps
	if fa, ok := raw["featuredApps"].([]interface{}); ok {
		for _, v := range fa {
			if s, ok := v.(string); ok {
				out.FeaturedApps = append(out.FeaturedApps, sanitizeString(s))
			}
		}
	}

	// apps
	if appsRaw, ok := raw["apps"].([]interface{}); ok {
		for _, a := range appsRaw {
			if am, ok := a.(map[string]interface{}); ok {
				app := App{}
				app.Name = getStr(am, "name")
				app.BundleIdentifier = getStr(am, "bundleIdentifier")
				app.DeveloperName = getStr(am, "developerName")
				app.Subtitle = getStr(am, "subtitle")
				app.LocalizedDescription = getStr(am, "localizedDescription")
				app.IconURL = getStr(am, "iconURL")
				app.TintColor = getStr(am, "tintColor")
				app.Category = getStr(am, "category")

				// screenshot handling:
				// prefer explicit screenshotURLs, but if absent, convert screenshots -> screenshotURLs
				if sUrls, ok := am["screenshotURLs"]; ok {
					if arr, ok := sUrls.([]interface{}); ok {
						for _, item := range arr {
							switch it := item.(type) {
							case string:
								app.ScreenshotURLs = append(app.ScreenshotURLs, sanitizeString(it))
							case map[string]interface{}:
								// try imageURL or url
								if s := getStr(it, "imageURL"); s != "" {
									app.ScreenshotURLs = append(app.ScreenshotURLs, s)
								} else if s := getStr(it, "url"); s != "" {
									app.ScreenshotURLs = append(app.ScreenshotURLs, s)
								}
							}
						}
					}
				} else if shots, ok := am["screenshots"]; ok {
					if arr, ok := shots.([]interface{}); ok {
						for _, item := range arr {
							switch itm := item.(type) {
							case string:
								app.ScreenshotURLs = append(app.ScreenshotURLs, sanitizeString(itm))
							case map[string]interface{}:
								// try "imageURL" or "url"
								if s := getStr(itm, "imageURL"); s != "" {
									app.ScreenshotURLs = append(app.ScreenshotURLs, s)
								} else if s := getStr(itm, "url"); s != "" {
									app.ScreenshotURLs = append(app.ScreenshotURLs, s)
								}
							}
						}
					}
				}

				// versions: convert date → UTC RFC3339, ignore buildVersion
				if versionsRaw, ok := am["versions"].([]interface{}); ok {
					for _, vr := range versionsRaw {
						if vm, ok := vr.(map[string]interface{}); ok {
							v := Version{
								Version:              getStr(vm, "version"),
								LocalizedDescription: getStr(vm, "localizedDescription"),
								DownloadURL:          getStr(vm, "downloadURL"),
								MinOSVersion:         getStr(vm, "minOSVersion"),
							}
							// date normalization
							if dateStr := getStr(vm, "date"); dateStr != "" {
								if parsed := parseFlexibleTime(dateStr); !parsed.IsZero() {
									v.Date = parsed.UTC().Format(time.RFC3339)
								} else {
									v.Date = dateStr
								}
							}
							// size normalization
							if sizeV, ok := vm["size"]; ok {
								switch n := sizeV.(type) {
								case float64:
									v.Size = int64(n)
								case int:
									v.Size = int64(n)
								case int64:
									v.Size = n
								}
							}
							app.Versions = append(app.Versions, v)
						}
					}
				}

				// preserve appPermissions as raw JSON if present
				if ap, ok := am["appPermissions"]; ok {
					if rawBytes, err := json.Marshal(ap); err == nil {
						app.AppPermissions = json.RawMessage(rawBytes)
					}
				}

				// explicitly skip marketplaceID, patreon, buildVersion by not copying them

				out.Apps = append(out.Apps, app)
			}
		}
	}

	// news: copy and normalize dates
	if newsRaw, ok := raw["news"].([]interface{}); ok {
		for _, n := range newsRaw {
			if nm, ok := n.(map[string]interface{}); ok {
				ni := NewsItem{
					Title:      getStr(nm, "title"),
					Identifier: getStr(nm, "identifier"),
					Caption:    getStr(nm, "caption"),
					TintColor:  getStr(nm, "tintColor"),
					ImageURL:   getStr(nm, "imageURL"),
					URL:        getStr(nm, "url"),
				}
				// notify may be bool
				if nb, ok := nm["notify"].(bool); ok {
					ni.Notify = nb
				}
				// appID could be null or a string - preserve whatever value was
				if v, ok := nm["appID"]; ok {
					ni.AppID = v
				}
				if dateRaw, ok := nm["date"].(string); ok && dateRaw != "" {
					if parsed := parseFlexibleTime(dateRaw); !parsed.IsZero() {
						ni.Date = parsed.UTC().Format(time.RFC3339)
					} else {
						ni.Date = dateRaw
					}
				}
				out.News = append(out.News, ni)
			}
		}
	}

	// marshal with indentation
	outBytes, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(4)
	}

	// write to output.json
	if err := ioutil.WriteFile("output.json", outBytes, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(5)
	}
	fmt.Println("Wrote output.json (ordered, normalized).")
}

func defaultIfEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// sanitizeString ensures we return a string with valid UTF-8 (replace bad bytes)
func sanitizeString(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return replaceInvalidUTF8(s)
}

// replaceInvalidUTF8 decodes runes, replacing invalid sequences with RuneError
func replaceInvalidUTF8(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// invalid single byte sequence -> append Unicode replacement char
			b.WriteRune(utf8.RuneError)
			i++
		} else {
			b.WriteRune(r)
			i += size
		}
	}
	return b.String()
}

// try multiple layouts to parse loosely formatted timestamps
func parseFlexibleTime(s string) time.Time {
	// trim spaces
	s = strings.TrimSpace(s)

	layouts := []string{
		time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",       // explicit Z (rare)
		"2006-01-02T15:04:05",        // no zone
		"2006-01-02T15:04",           // minutes only
		"2006-01-02 15:04:05",        // space separator
		"2006-01-02",                 // date only
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	// Try parsing with timezone offset omitted but assume local
	if t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local); err == nil {
		return t
	}
	return time.Time{}
}

