package structure

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bridgecrewio/yor/src/common/structure"
	"github.com/bridgecrewio/yor/src/common/tagging/simple"
	"github.com/bridgecrewio/yor/src/common/tagging/tags"
	"github.com/bridgecrewio/yor/src/common/yaml"
	"github.com/stretchr/testify/assert"
)

func TestCloudformationParser_ParseFile(t *testing.T) {
	t.Run("parse ebs file", func(t *testing.T) {
		directory := "../../../tests/cloudformation/resources/ebs"
		cfnParser := CloudformationParser{}
		cfnParser.Init(directory, nil)
		cfnBlocks, err := cfnParser.ParseFile(directory + "/ebs.yaml")
		if err != nil {
			t.Errorf("ParseFile() error = %v", err)
			return
		}
		assert.Equal(t, 1, len(cfnBlocks))
		newVolumeBlock := cfnBlocks[0]
		assert.Equal(t, structure.Lines{Start: 3, End: 13}, newVolumeBlock.GetLines())
		assert.Equal(t, "NewVolume", newVolumeBlock.GetResourceID())

		existingTag := newVolumeBlock.GetExistingTags()[0]
		assert.Equal(t, "MyTag", existingTag.GetKey())
		assert.Equal(t, "TagValue", existingTag.GetValue())
		assert.Equal(t, 3, cfnParser.FileToResourcesLines[directory+"/ebs.yaml"].Start)
		assert.Equal(t, 13, cfnParser.FileToResourcesLines[directory+"/ebs.yaml"].End)
	})

	t.Run("parse_simple_template", func(t *testing.T) {
		directory, _ := filepath.Abs("../../../tests/cloudformation/resources/no_tags")
		cfnParser := CloudformationParser{}
		cfnParser.Init(directory, nil)
		sourceFile := directory + "/base.template"
		cfnBlocks, _ := cfnParser.ParseFile(sourceFile)
		assert.Equal(t, 1, len(cfnBlocks))
		assert.Equal(t, 2, cfnParser.FileToResourcesLines[sourceFile].Start)
		assert.Equal(t, 9, cfnParser.FileToResourcesLines[sourceFile].End)
	})
}

func compareLines(t *testing.T, expected map[string]*structure.Lines, actual map[string]*structure.Lines) {
	for resourceName := range expected {
		actualLines := actual[resourceName]
		if actualLines == nil {
			t.Errorf("expected %s to be in resources mapping", resourceName)
		}
		expctedLines := expected[resourceName]
		assert.Equal(t, expctedLines, actualLines)
	}
}

func Test_mapResourcesLineYAML(t *testing.T) {
	t.Run("test single resource", func(t *testing.T) {
		filePath := "../../../tests/cloudformation/resources/ebs/ebs.yaml"
		resourcesNames := []string{"NewVolume"}
		expected := map[string]*structure.Lines{
			"NewVolume": {Start: 3, End: 13},
		}
		actual := yaml.MapResourcesLineYAML(filePath, resourcesNames, ResourcesStartToken)
		compareLines(t, expected, actual)
	})

	t.Run("test multiple resources", func(t *testing.T) {
		filePath := "../../../tests/cloudformation/resources/ec2_untagged/ec2_untagged.yaml"
		resourcesNames := []string{"EC2InstanceResource0", "EC2InstanceResource1", "EC2LaunchTemplateResource0", "EC2LaunchTemplateResource1"}
		expected := map[string]*structure.Lines{
			"EC2InstanceResource0":       {Start: 2, End: 5},
			"EC2InstanceResource1":       {Start: 6, End: 15},
			"EC2LaunchTemplateResource0": {Start: 16, End: 20},
			"EC2LaunchTemplateResource1": {Start: 21, End: 31},
		}
		actual := yaml.MapResourcesLineYAML(filePath, resourcesNames, ResourcesStartToken)
		compareLines(t, expected, actual)
	})

	t.Run("test CFN writing", func(t *testing.T) {
		directory := "../../../tests/cloudformation/resources/ebs"
		f, _ := ioutil.TempFile(directory, "temp.*.yaml")
		cfnParser := CloudformationParser{}
		cfnParser.Init(directory, nil)
		readFilePath := directory + "/ebs.yaml"
		tagGroup := simple.TagGroup{}
		extraTags := []tags.ITag{
			&tags.Tag{
				Key:   "new_tag",
				Value: "new_value",
			},
		}
		tagGroup.SetTags(extraTags)
		tagGroup.InitTagGroup("", []string{})
		writeFilePath := directory + "/ebs_tagged.yaml"
		cfnBlocks, err := cfnParser.ParseFile(readFilePath)
		for _, block := range cfnBlocks {
			err := tagGroup.CreateTagsForBlock(block)
			if err != nil {
				t.Fail()
			}
		}
		if err != nil {
			t.Fail()
		}
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			t.Fail()
		}
		err = cfnParser.WriteFile(readFilePath, cfnBlocks, f.Name())
		if err != nil {
			t.Fail()
		}
		var expectedHandler, actualHandler *os.File
		expectedAbs, _ := filepath.Abs(writeFilePath)
		actualAbs, _ := filepath.Abs(f.Name())
		expectedHandler, _ = os.OpenFile(expectedAbs, os.O_RDWR, 0755)
		actualHandler, _ = os.OpenFile(actualAbs, os.O_RDWR|os.O_CREATE, 0755)
		_, err = expectedHandler.Seek(0, io.SeekStart)
		if err != nil {
			t.Fail()
		}
		_, err = actualHandler.Seek(0, io.SeekStart)
		if err != nil {
			t.Fail()
		}
		defer func() {
			_ = expectedHandler.Close()
			_ = actualHandler.Close()
			_ = os.Remove(f.Name())
		}()
		actualReader := bufio.NewScanner(actualHandler)
		expectedReader := bufio.NewScanner(expectedHandler)
		for actualReader.Scan() && expectedReader.Scan() {
			actualLine := actualReader.Text()
			expectedLine := expectedReader.Text()
			assert.Equal(t, strings.Trim(actualLine, " \n\t"), strings.Trim(expectedLine, " \n\t"))
		}
	})
}
