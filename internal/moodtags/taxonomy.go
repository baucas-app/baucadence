package moodtags

import "strings"

// Mood categories a raw Last.fm folksonomy tag can be classified into.
// Genre-only tags (e.g. "rock", "90s") simply have no entry and are ignored.
const (
	MoodHappy       = "happy"
	MoodChill       = "chill"
	MoodMelancholic = "melancholic"
	MoodEnergetic   = "energetic"
	MoodAggressive  = "aggressive"
	MoodRomantic    = "romantic"
	MoodDark        = "dark"
	MoodPeaceful    = "peaceful"
)

// tagToMood maps lowercased Last.fm tags to a mood category. Not exhaustive -
// it only needs to cover the tags people actually apply often enough to move
// an aggregate score.
var tagToMood = map[string]string{
	// happy / upbeat
	"happy":       MoodHappy,
	"happy music": MoodHappy,
	"feel good":   MoodHappy,
	"feelgood":    MoodHappy,
	"upbeat":      MoodHappy,
	"fun":         MoodHappy,
	"cheerful":    MoodHappy,
	"summer":      MoodHappy,
	"sunny":       MoodHappy,
	"good vibes":  MoodHappy,
	"joyful":      MoodHappy,
	"feel-good":   MoodHappy,

	// chill / mellow
	"chill":          MoodChill,
	"chillout":       MoodChill,
	"chill out":      MoodChill,
	"mellow":         MoodChill,
	"laid back":      MoodChill,
	"laid-back":      MoodChill,
	"relax":          MoodChill,
	"lounge":         MoodChill,
	"dreamy":         MoodChill,
	"dream pop":      MoodChill,
	"ambient":        MoodChill,
	"soft":           MoodChill,
	"easy listening": MoodChill,

	// melancholic / sad
	"sad":         MoodMelancholic,
	"sadcore":     MoodMelancholic,
	"sadness":     MoodMelancholic,
	"melancholy":  MoodMelancholic,
	"melancholic": MoodMelancholic,
	"bittersweet": MoodMelancholic,
	"heartbreak":  MoodMelancholic,
	"crying":      MoodMelancholic,
	"lonely":      MoodMelancholic,
	"depressing":  MoodMelancholic,
	"melancholia": MoodMelancholic,

	// energetic / hype
	"energetic":    MoodEnergetic,
	"energy":       MoodEnergetic,
	"hype":         MoodEnergetic,
	"party":        MoodEnergetic,
	"dance":        MoodEnergetic,
	"upbeat tempo": MoodEnergetic,
	"driving":      MoodEnergetic,
	"workout":      MoodEnergetic,
	"anthem":       MoodEnergetic,
	"pump up":      MoodEnergetic,

	// aggressive / angry
	"aggressive": MoodAggressive,
	"angry":      MoodAggressive,
	"anger":      MoodAggressive,
	"rage":       MoodAggressive,
	"brutal":     MoodAggressive,
	"intense":    MoodAggressive,
	"heavy":      MoodAggressive,
	"harsh":      MoodAggressive,

	// romantic
	"romantic":   MoodRomantic,
	"love":       MoodRomantic,
	"love songs": MoodRomantic,
	"sensual":    MoodRomantic,
	"sexy":       MoodRomantic,

	// dark / moody
	"dark":         MoodDark,
	"moody":        MoodDark,
	"gloomy":       MoodDark,
	"eerie":        MoodDark,
	"haunting":     MoodDark,
	"brooding":     MoodDark,
	"dark ambient": MoodDark,

	// peaceful / calm
	"calm":       MoodPeaceful,
	"calming":    MoodPeaceful,
	"peaceful":   MoodPeaceful,
	"soothing":   MoodPeaceful,
	"serene":     MoodPeaceful,
	"gentle":     MoodPeaceful,
	"meditative": MoodPeaceful,
}

// Categorize maps a raw Last.fm tag to a mood category. It returns "" if the
// tag isn't a recognized mood tag (e.g. it's a genre or decade tag).
func Categorize(tag string) string {
	return tagToMood[strings.ToLower(strings.TrimSpace(tag))]
}
