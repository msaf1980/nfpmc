package main

import (
	"os"
	"path"
	"runtime"
	"testing"

	_ "github.com/goreleaser/nfpm/v2/apk"
	_ "github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
	_ "github.com/goreleaser/nfpm/v2/rpm"

	"github.com/stretchr/testify/assert"
)

func Test_rewriteFileName(t *testing.T) {
	tests := []struct {
		name     string
		filesMap StringMap
		want     string
		wantErr  bool
	}{
		{
			name:     "out/usr/bin/test",
			filesMap: StringMap{"out/": "/", "conf/": "/etc/", "doc/": "/usr/share/test"},
			want:     "/usr/bin/test",
			wantErr:  false,
		},
		{
			name:     "conf/test.conf",
			filesMap: StringMap{"out/": "/", "conf/": "/etc/", "doc/": "/usr/share/test"},
			want:     "/etc/test.conf",
			wantErr:  false,
		},
		{
			name:     "none/usr/bin/test",
			filesMap: StringMap{"out/": "/", "conf/": "/etc/", "doc/": "/usr/share/test"},
			want:     "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewriteFileName(tt.name, tt.filesMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("rewriteFileName(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("rewriteFileName(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPackage(t *testing.T) {
	// /d, _ := diff.NewDiffer(diff.TagName("json"))

	_, filename, _, _ := runtime.Caller(0)
	testDir := path.Join(path.Dir(path.Dir(path.Dir(filename))), "test")

	var p Packager

	err := os.Chdir(testDir)
	assert.NoErrorf(t, err, "Chdir")

	p.Info.Name = "test"
	p.Info.Version = "1.0.0"
	p.Info.Release = "1"

	err = p.Init()
	assert.NoErrorf(t, err, "Package.Init")

	// Init files list
	verifyContent := files.Contents{
		&files.Content{Source: "out/test-example", Destination: "/usr/bin/test-example", Type: defaultStr},
		// &files.Content{Source: "conf/test-example.conf", Destination: "/etc/test-example.conf", Type: defaultStr},
		// &files.Content{Source: "docs/test-example.txt", Destination: "/usr/share/test/test-example.txt", Type: defaultStr},
		&files.Content{Source: "conf/", Destination: "/etc/", Type: defaultStr},
		&files.Content{Source: "docs/", Destination: "/usr/share/test/", Type: defaultStr},
	}
	err = p.AddFiles([]string{"out/test-example=/usr/bin/test-example", "conf/=/etc/", "docs/=/usr/share/test/"})
	assert.NoErrorf(t, err, "Package.AddFiles")
	assert.Equalf(t, verifyContent, p.Info.Contents, "Package.AddFiles Contents mismatch")

	// Add symlink
	verifyContent = files.Contents{
		&files.Content{Source: "out/test-example", Destination: "/usr/bin/test-example", Type: defaultStr},
		// &files.Content{Source: "conf/test-example.conf", Destination: "/etc/test-example.conf", Type: defaultStr},
		// &files.Content{Source: "docs/test-example.txt", Destination: "/usr/share/test/test-example.txt", Type: defaultStr},
		&files.Content{Source: "conf/", Destination: "/etc/", Type: defaultStr},
		&files.Content{Source: "docs/", Destination: "/usr/share/test/", Type: defaultStr},
		&files.Content{Source: "/usr/bin/test-example", Destination: "/usr/bin/test-link", Type: symlinkStr},
	}
	err = p.AddSymlinks([]string{"/usr/bin/test-example=/usr/bin/test-link"})
	assert.NoErrorf(t, err, "Package.AddSymlinks")
	assert.Equalf(t, verifyContent, p.Info.Contents, "Package.AddSymlinks Contents mismatch")

	// Add duplicate symlink, must failed
	err = p.AddSymlinks([]string{"/usr/bin/test-noexist=/usr/bin/test-link"})
	if err == nil {
		t.Errorf("Package.AddSymlinks success, but already exist\n")
	}

	// Set config
	verifyContent = files.Contents{
		&files.Content{Source: "out/test-example", Destination: "/usr/bin/test-example", Type: defaultStr},
		// &files.Content{Source: "conf/test-example.conf", Destination: "/etc/test-example.conf", Type: configStr},
		// &files.Content{Source: "docs/test-example.txt", Destination: "/usr/share/test/test-example.txt", Type: defaultStr},
		&files.Content{Source: "conf/", Destination: "/etc/", Type: configStr},
		&files.Content{Source: "docs/", Destination: "/usr/share/test/", Type: defaultStr},
		&files.Content{Source: "/usr/bin/test-example", Destination: "/usr/bin/test-link", Type: symlinkStr},
	}
	err = p.SetConfigFiles([]string{"/etc"})
	assert.NoErrorf(t, err, "Package.SetConfigFiles")
	assert.Equalf(t, verifyContent, p.Info.Contents, "Package.SetConfigFiles Contents mismatch")

	// Set doc for config, must failed
	err = p.SetDocFiles([]string{"/etc/test-example.conf"})
	assert.NoErrorf(t, err, "Package.SetDocsFiles success, but already exist")

	// Set config
	verifyContent = files.Contents{
		&files.Content{Source: "out/test-example", Destination: "/usr/bin/test-example", Type: defaultStr},
		// &files.Content{Source: "conf/test-example.conf", Destination: "/etc/test-example.conf", Type: configStr},
		// &files.Content{Source: "docs/test-example.txt", Destination: "/usr/share/test/test-example.txt", Type: docStr},
		&files.Content{Source: "conf/", Destination: "/etc/", Type: configStr},
		&files.Content{Source: "docs/", Destination: "/usr/share/test/", Type: docStr},
		&files.Content{Source: "/usr/bin/test-example", Destination: "/usr/bin/test-link", Type: symlinkStr},
	}
	err = p.SetDocFiles([]string{"/usr/share/test/test-example.txt"})
	assert.NoErrorf(t, err, "Package.SetDocFiles")

	assert.Equalf(t, verifyContent, p.Info.Contents, "Package.SetDocFiles Contents mismatch")
}
