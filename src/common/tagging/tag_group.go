package tagging

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bridgecrewio/yor/src/common/logger"
	"github.com/bridgecrewio/yor/src/common/structure"
	"github.com/bridgecrewio/yor/src/common/tagging/tags"
)

type TagGroup struct {
	tags        []tags.ITag
	SkippedTags []string
}

var IgnoredDirs = []string{".git", ".DS_Store", ".idea"}

type ITagGroup interface {
	InitTagGroup(path string, skippedTags []string)
	CreateTagsForBlock(block structure.IBlock) error
	GetTags() []tags.ITag
	GetDefaultTags() []tags.ITag
}

func (t *TagGroup) GetSkippedDirs() []string {
	return IgnoredDirs
}

func (t *TagGroup) SetTags(tags []tags.ITag) {
	for _, tag := range tags {
		tag.Init()
		if !t.IsTagSkipped(tag) {
			t.tags = append(t.tags, tag)
		}
	}
}

func (t *TagGroup) GetTags() []tags.ITag {
	return t.tags
}

func (t *TagGroup) IsTagSkipped(tag tags.ITag) bool {
	for _, st := range t.SkippedTags {
		stRegex := strings.ReplaceAll(st, "*", ".*")
		if match, err := regexp.Match(stRegex, []byte(tag.GetKey())); match || err != nil {
			logger.Info(fmt.Sprintf("Skipping %v due to skip-tag constraint %v", tag.GetKey(), st))
			return true
		}
	}
	return false
}

func (t *TagGroup) UpdateBlockTags(block structure.IBlock, data interface{}) error {
	var newTags []tags.ITag
	var err error
	var tagVal tags.ITag
	for _, tag := range t.GetTags() {
		tagVal, err = tag.CalculateValue(data)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create %v tag for block %v", tag.GetKey(), block.GetResourceID()))
		}
		newTags = append(newTags, tagVal)
	}
	block.AddNewTags(newTags)
	return err
}
