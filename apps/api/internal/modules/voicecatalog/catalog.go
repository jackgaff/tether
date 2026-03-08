package voicecatalog

import (
	"fmt"
	"slices"
)

type Voice struct {
	ID                     string `json:"id"`
	DisplayName            string `json:"displayName"`
	Locale                 string `json:"locale"`
	Polyglot               bool   `json:"polyglot"`
	IsDefault              bool   `json:"isDefault"`
	BrowserSupported       bool   `json:"browserSupported"`
	ConnectNativeSupported bool   `json:"connectNativeSupported"`
}

type Catalog struct {
	defaultVoiceID string
	allowed        []Voice
	allowedByID    map[string]Voice
}

func New(defaultVoiceID string, allowedVoiceIDs []string) (Catalog, error) {
	if len(allowedVoiceIDs) == 0 {
		return Catalog{}, fmt.Errorf("at least one allowed voice is required")
	}

	voices := make([]Voice, 0, len(allowedVoiceIDs))
	allowedByID := make(map[string]Voice, len(allowedVoiceIDs))

	for _, voiceID := range allowedVoiceIDs {
		voice, ok := knownVoices[voiceID]
		if !ok {
			return Catalog{}, fmt.Errorf("unknown voice id %q", voiceID)
		}

		voice.IsDefault = voiceID == defaultVoiceID
		voices = append(voices, voice)
		allowedByID[voiceID] = voice
	}

	if !slices.Contains(allowedVoiceIDs, defaultVoiceID) {
		return Catalog{}, fmt.Errorf("default voice %q is not allowed", defaultVoiceID)
	}

	return Catalog{
		defaultVoiceID: defaultVoiceID,
		allowed:        voices,
		allowedByID:    allowedByID,
	}, nil
}

func (c Catalog) Allowed() []Voice {
	return append([]Voice(nil), c.allowed...)
}

func (c Catalog) DefaultVoiceID() string {
	return c.defaultVoiceID
}

func (c Catalog) Resolve(id string) (Voice, bool) {
	voice, ok := c.allowedByID[id]
	return voice, ok
}

func (c Catalog) IsAllowed(id string) bool {
	_, ok := c.Resolve(id)
	return ok
}

func KnownIDs() []string {
	return append([]string(nil), knownVoiceOrder...)
}

var knownVoices = map[string]Voice{
	"tiffany":  {ID: "tiffany", DisplayName: "Tiffany", Locale: "en-US", Polyglot: true, BrowserSupported: true, ConnectNativeSupported: false},
	"matthew":  {ID: "matthew", DisplayName: "Matthew", Locale: "en-US", Polyglot: true, BrowserSupported: true, ConnectNativeSupported: true},
	"amy":      {ID: "amy", DisplayName: "Amy", Locale: "en-GB", BrowserSupported: true, ConnectNativeSupported: true},
	"olivia":   {ID: "olivia", DisplayName: "Olivia", Locale: "en-AU", BrowserSupported: true, ConnectNativeSupported: true},
	"kiara":    {ID: "kiara", DisplayName: "Kiara", Locale: "en-IN / hi-IN", BrowserSupported: true, ConnectNativeSupported: false},
	"arjun":    {ID: "arjun", DisplayName: "Arjun", Locale: "en-IN / hi-IN", BrowserSupported: true, ConnectNativeSupported: false},
	"ambre":    {ID: "ambre", DisplayName: "Ambre", Locale: "fr-FR", BrowserSupported: true, ConnectNativeSupported: false},
	"florian":  {ID: "florian", DisplayName: "Florian", Locale: "fr-FR", BrowserSupported: true, ConnectNativeSupported: false},
	"beatrice": {ID: "beatrice", DisplayName: "Beatrice", Locale: "it-IT", BrowserSupported: true, ConnectNativeSupported: false},
	"lorenzo":  {ID: "lorenzo", DisplayName: "Lorenzo", Locale: "it-IT", BrowserSupported: true, ConnectNativeSupported: false},
	"tina":     {ID: "tina", DisplayName: "Tina", Locale: "de-DE", BrowserSupported: true, ConnectNativeSupported: false},
	"lennart":  {ID: "lennart", DisplayName: "Lennart", Locale: "de-DE", BrowserSupported: true, ConnectNativeSupported: false},
	"lupe":     {ID: "lupe", DisplayName: "Lupe", Locale: "es-US", BrowserSupported: true, ConnectNativeSupported: true},
	"carlos":   {ID: "carlos", DisplayName: "Carlos", Locale: "es-US", BrowserSupported: true, ConnectNativeSupported: false},
	"carolina": {ID: "carolina", DisplayName: "Carolina", Locale: "pt-BR", BrowserSupported: true, ConnectNativeSupported: false},
	"leo":      {ID: "leo", DisplayName: "Leo", Locale: "pt-BR", BrowserSupported: true, ConnectNativeSupported: false},
}

var knownVoiceOrder = []string{
	"tiffany",
	"matthew",
	"amy",
	"olivia",
	"kiara",
	"arjun",
	"ambre",
	"florian",
	"beatrice",
	"lorenzo",
	"tina",
	"lennart",
	"lupe",
	"carlos",
	"carolina",
	"leo",
}
