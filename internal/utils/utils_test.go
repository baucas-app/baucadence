package utils_test

import (
	"testing"

	"github.com/gabehf/koito/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestRemoveNonAscii(t *testing.T) {
	expected := [][]string{
		[]string{"test1", "test2"}, []string{"test1", "test2"},
		[]string{"ネクライトーキー", "NECRY TALKIE"}, []string{"NECRY TALKIE"},
		[]string{"BFY#& cn&W,KE|"}, []string{"BFY#& cn&W,KE|"},
		[]string{"もっさ"}, []string{},
	}

	for i := 0; i < len(expected)/2; i = i + 2 {
		r := utils.RemoveNonAscii(expected[i])
		assert.EqualValues(t, expected[i+1], r)
	}
}

func TestUniqueIgnoringCase(t *testing.T) {
	expected := [][]string{
		[]string{"Necry Talkie", "NECRY TALKIE"}, []string{"Necry Talkie"},
		[]string{"ネクライトーキー", "NECRY TALKIE"}, []string{"ネクライトーキー", "NECRY TALKIE"},
		[]string{"BFY#& cn&W,KE|"}, []string{"BFY#& cn&W,KE|"},
		[]string{"もっさ"}, []string{"もっさ"},
	}

	for i := 0; i < len(expected)/2; i = i + 2 {
		r := utils.UniqueIgnoringCase(expected[i])
		assert.EqualValues(t, expected[i+1], r)
	}
}

func TestRemoveInBoth(t *testing.T) {
	expected := [][]string{
		{"Necry Talkie", "NECRY TALKIE"}, {"Necry Talkie"}, {"NECRY TALKIE"},
		{"ネクライトーキー", "NECRY TALKIE"}, {"ネクライトーキー", "NECRY TALKIE"}, {},
		{"BFY#& cn&W,KE|", "bleh"}, {"BFY#& cn&W,KE|"}, {"bleh"},
	}

	for i := 0; i < len(expected)/3; i = i + 3 {
		r := utils.RemoveInBoth(expected[i], expected[i+1])
		assert.EqualValues(t, expected[i+2], r)
	}
}
