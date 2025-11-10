// fixrepo.go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const (
	hardcodedIdentifier = "com.ripestore.source"
	hardcodedSourceURL  = "https://raw.githubusercontent.com/RipeStore/repos/main/RipeStore_feather.json"
)

// Root has fields in the order we want them to appear in output JSON.
type Root struct {
	Name         string        `json:"name,omitempty"`
	Subtitle     string        `json:"subtitle,omitempty"`
	Identifier   string        `json:"identifier,omitempty"`
	SourceURL    string        `json:"sourceURL,omitempty"`
	Description  string        `json:"description,omitempty"`
	IconURL      string        `json:"iconURL,omitempty"`
	Website      string        `json:"website,omitempty"`
	PatreonURL   string        `json:"patreonURL,omitempty"`
	TintColor    string        `json:"tintColor,omitempty"`
	FeaturedApps []string      `json:"featuredApps,omitempty"`
	Apps         []App         `json:"apps,omitempty"`
	News         []NewsItem    `json:"news,omitempty"`
}

// App defines fields and order for each app entry.
type App struct {
	Name                string            `json:"name,omitempty"`
	BundleIdentifier    string            `json:"bundleIdentifier,omitempty"`
	DeveloperName       string            `json:"developerName,omitempty"`
	Subtitle            string            `json:"subtitle,omitempty"`
	LocalizedDescription string           `json:"localizedDescription,omitempty"`
	IconURL             string            `json:"iconURL,omitempty"`
	TintColor           string            `json:"tintColor,omitempty"`
	Category            string            `json:"category,omitempty"`
	ScreenshotURLs      []string          `json:"screenshotURLs,omitempty"`
	Versions            []Version         `json:"versions,omitempty"`
	AppPermissions      json.RawMessage   `json:"appPermissions,omitempty"` // preserve as-is
	// intentionally omitting marketplaceID, patreon, buildVersion
}

// Version defines version object order.
type Version struct {
	Version              string `json:"version,omitempty"`
	Date                 string `json:"date,omitempty"`
	LocalizedDescription string `json:"localizedDescription,omitempty"`
	DownloadURL          string `json:"downloadURL,omitempty"`
	Size                 int64  `json:"size,omitempty"`
	MinOSVersion         string `json:"minOSVersion,omitempty"`
	// buildVersion intentionally omitted (removed)
}

// NewsItem with ordered fields
type NewsItem struct {
	Title    string `json:"title,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Date     string `json:"date,omitempty"`
	TintColor string `json:"tintColor,omitempty"`
	ImageURL string `json:"imageURL,omitempty"`
	Notify   bool   `json:"notify,omitempty"`
	URL      string `json:"url,omitempty"`
	AppID    interface{} `json:"appID,omitempty"`
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

	// Unmarshal into a generic map first to be flexible with incoming shape
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		fmt.Fprintln(os.Stderr, "json parse:", err)
		os.Exit(3)
	}

	out := Root{}

	// helper to extract string safely
	getStr := func(m map[string]interface{}, k string) string {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	out.Name = getStr(raw, "name")
	out.Subtitle = getStr(raw, "subtitle")

	// identifier & sourceURL: keep existing, else hardcode
	if id := getStr(raw, "identifier"); id != "" {
		out.Identifier = id
	} else {
		out.Identifier = hardcodedIdentifier
	}
	if su := getStr(raw, "sourceURL"); su != "" {
		out.SourceURL = su
	} else {
		out.SourceURL = hardcodedSourceURL
	}

	out.Description = getStr(raw, "description")
	out.IconURL = getStr(raw, "iconURL")
	out.Website = getStr(raw, "website")
	out.PatreonURL = getStr(raw, "patreonURL")
	out.TintColor = getStr(raw, "tintColor")

	// featuredApps
	if fa, ok := raw["featuredApps"].([]interface{}); ok {
		for _, v := range fa {
			if s, ok := v.(string); ok {
				out.FeaturedApps = append(out.FeaturedApps, s)
			}
		}
	}

	// apps: iterate original order, map fields into App struct while removing unwanted fields
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
				// - if "screenshotURLs" exists and is slice of strings, copy
				// - else if "screenshots" is []string or []object { imageURL: ... }, convert to []string
				if sUrls, ok := am["screenshotURLs"]; ok {
					// may be []interface{} of strings
					if arr, ok := sUrls.([]interface{}); ok {
						for _, item := range arr {
							if s, ok := item.(string); ok {
								app.ScreenshotURLs = append(app.ScreenshotURLs, s)
							}
						}
					}
				} else if shots, ok := am["screenshots"]; ok {
					if arr, ok := shots.([]interface{}); ok {
						for _, item := range arr {
							switch itm := item.(type) {
							case string:
								app.ScreenshotURLs = append(app.ScreenshotURLs, itm)
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

				// versions: copy, omit buildVersion
				if versionsRaw, ok := am["versions"].([]interface{}); ok {
					for _, vr := range versionsRaw {
						if vm, ok := vr.(map[string]interface{}); ok {
							v := Version{
								Version:              getStr(vm, "version"),
								Date:                 getStr(vm, "date"),
								LocalizedDescription: getStr(vm, "localizedDescription"),
								DownloadURL:          getStr(vm, "downloadURL"),
								MinOSVersion:         getStr(vm, "minOSVersion"),
							}
							// size might be number
							if sizeV, ok := vm["size"]; ok {
								switch n := sizeV.(type) {
								case float64:
									v.Size = int64(n)
								case int64:
									v.Size = n
								case int:
									v.Size = int64(n)
								}
							}
							// don't copy buildVersion (intentionally removed)
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

				// we intentionally skip marketplaceID and patreon keys by not copying them

				out.Apps = append(out.Apps, app)
			}
		}
	}

	// news: copy and parse/normalize date
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
				// parse date using some common layouts, then format as UTC RFC3339 (with Z)
				if dateRaw, ok := nm["date"].(string); ok && dateRaw != "" {
					if parsed := parseFlexibleTime(dateRaw); !parsed.IsZero() {
						ni.Date = parsed.UTC().Format("2006-01-02T15:04:05Z")
					} else {
						// if we couldn't parse, just keep original
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
	fmt.Println("Wrote output.json (ordered).")
}

// try multiple layouts to parse loosely formatted timestamps
func parseFlexibleTime(s string) time.Time {
	layouts := []string{
		time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05Z",       // explicit Z
		"2006-01-02T15:04:05",        // no zone
		"2006-01-02T15:04",           // minutes only
		"2006-01-02 15:04:05",        // space separator
		"2006-01-02",                 // date only
		time.RFC3339Nano,
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
