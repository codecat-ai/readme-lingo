package lingo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const (
	metadataPrefix = "<!-- readme-lingo:"
	metadataSuffix = "-->"
	switcherStart  = "<!-- readme-lingo-switcher:start -->"
	switcherEnd    = "<!-- readme-lingo-switcher:end -->"
)

type Metadata struct {
	SourcePath  string    `json:"source"`
	Target      string    `json:"target"`
	Model       string    `json:"model"`
	SourceHash  string    `json:"digest"`
	GeneratedAt time.Time `json:"generated"`
}

func SourceDigest(source []byte) string {
	sum := sha256.Sum256(source)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func AddMetadata(markdown string, meta Metadata) string {
	payload, _ := json.Marshal(meta)
	markdown = strings.TrimRight(markdown, "\n")
	return markdown + "\n\n" + metadataPrefix + " " + string(payload) + " " + metadataSuffix + "\n"
}

func ParseMetadata(markdown []byte) (Metadata, bool) {
	text := string(markdown)
	idx := strings.LastIndex(text, metadataPrefix)
	if idx < 0 {
		return Metadata{}, false
	}
	rest := text[idx+len(metadataPrefix):]
	end := strings.Index(rest, metadataSuffix)
	if end < 0 {
		return Metadata{}, false
	}
	payload := strings.TrimSpace(rest[:end])
	var meta Metadata
	if err := json.Unmarshal([]byte(payload), &meta); err != nil {
		return Metadata{}, false
	}
	return meta, true
}

func IsSynchronized(source []byte, translated []byte) bool {
	meta, ok := ParseMetadata(translated)
	return ok && meta.SourceHash == SourceDigest(source)
}

func BuildSwitcher(spec string) (string, error) {
	pairs := strings.Split(spec, ",")
	links := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		left, right, ok := strings.Cut(pair, ":")
		if !ok || strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
			return "", ErrInvalidSwitcher
		}
		label := strings.TrimSpace(left)
		path := strings.TrimSpace(right)
		links = append(links, "["+label+"]("+path+")")
	}
	if len(links) == 0 {
		return "", ErrInvalidSwitcher
	}
	return switcherStart + "\n" + strings.Join(links, " | ") + "\n" + switcherEnd, nil
}

func InsertSwitcher(markdown string, switcher string) string {
	markdown = strings.TrimLeft(markdown, "\n")
	if strings.HasPrefix(markdown, switcherStart) {
		end := strings.Index(markdown, switcherEnd)
		if end >= 0 {
			after := markdown[end+len(switcherEnd):]
			return switcher + "\n\n" + strings.TrimLeft(after, "\n")
		}
	}
	return switcher + "\n\n" + markdown
}
