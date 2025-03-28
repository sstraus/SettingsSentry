package config

import (
	"SettingsSentry/pkg/util"
	"os"
	"testing"
)

func TestGetICloudFolderLocationErrors(t *testing.T) {
	setupTestDependencies()

	originalGetHomeDirectory := GetHomeDirectory
	originalFs := util.Fs
	defer func() {
		GetHomeDirectory = originalGetHomeDirectory
		util.Fs = originalFs
	}()

	GetHomeDirectory = func() (string, error) {
		return "/mock/home", nil
	}

	mockFs := &mockFileSystem{
		statFunc: func(path string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		readFileFunc: nil,
	}
	util.Fs = mockFs

	_, iCloudErr := GetICloudFolderLocation()
	if iCloudErr == nil {
		t.Errorf("Expected error when iCloud folder doesn't exist, got nil")
	}
}
