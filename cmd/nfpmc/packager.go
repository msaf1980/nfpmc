package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/apk"
	_ "github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
	_ "github.com/goreleaser/nfpm/v2/rpm"
)

const (
	defaultStr = ""
	configStr  = "config|noreplace"
	symlinkStr = "symlink"
	docStr     = "doc"
)

type InputType uint8

const (
	INPUT_DIR InputType = iota
)

var inputTypeStr = []string{"dir"}

func (i *InputType) Set(value string) error {
	switch strings.ToLower(value) {
	case "dir":
		*i = INPUT_DIR
	default:
		return fmt.Errorf("unknown input type")
	}
	return nil
}

func (i *InputType) String() string {
	return inputTypeStr[*i]
}

func (i *InputType) Type() string {
	return "intput_type"
}

type OutputType uint8

const (
	RPM OutputType = iota
	DEB
	APK
)

var outputTypeStr = []string{"rpm", "deb", "apk"}

func (i *OutputType) Set(value string) error {
	switch strings.ToLower(value) {
	case "rpm":
		*i = RPM
	case "deb":
		*i = DEB
	case "apk":
		*i = APK
	default:
		return fmt.Errorf("unknown output type")
	}
	return nil
}

func (i *OutputType) String() string {
	return outputTypeStr[*i]
}

func (i *OutputType) Type() string {
	return "output_type"
}

type StringSlice []string

func (u *StringSlice) Set(value string) error {
	*u = append(*u, value)
	return nil
}

func (u *StringSlice) String() string {
	return "[ " + strings.Join(*u, ", ") + " ]"
}

func (u *StringSlice) Type() string {
	return "[]string"
}

type StringMap map[string]string

func (u *StringMap) Set(key, value string) error {
	(*u)[key] = value
	return nil
}

func (u *StringMap) String() string {
	var sb strings.Builder
	sb.WriteString("[ ")
	first := true
	for k, v := range *u {
		if first {
			first = false
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteString(" = ")
		sb.WriteString(v)
	}
	sb.WriteString(" ]")
	return sb.String()
}

func (u *StringMap) Type() string {
	return "map[string]string"
}

// type FileMap struct {
// 	Src string
// 	Dst string
// }

type FileContentMap map[string]*files.Content

type Packager struct {
	InputType  InputType
	OutputType OutputType
	OutDir     string
	OutName    string

	Info        nfpm.Info
	Compression string
	PostUpgrade string
	PreUpgrade  string

	FilesMap FileContentMap
}

func charsToString(ca []int8) string {
	s := make([]byte, len(ca))
	var lens int
	for ; lens < len(ca); lens++ {
		if ca[lens] == 0 {
			break
		}
		s[lens] = uint8(ca[lens])
	}
	return string(s[0:lens])
}

func (p *Packager) Init() error {
	if len(p.Info.Name) == 0 {
		return fmt.Errorf("name not set")
	}
	if len(p.Info.Version) == 0 {
		return fmt.Errorf("version not set")
	}
	if len(p.Info.Release) == 0 {
		return fmt.Errorf("iteration not set")
	}

	if p.Info.Arch == "" {
		var buf syscall.Utsname
		err := syscall.Uname(&buf)
		if err != nil {
			return err
		}
		arch := charsToString(buf.Machine[:])
		if arch == "x86_64" || arch == "amd64" {
			if p.OutputType == RPM {
				p.Info.Arch = "x86_64"
			} else {
				p.Info.Arch = "amd64"
			}
		}
	}

	p.FilesMap = make(FileContentMap)

	return nil
}

func (p *Packager) Validate() error {
	nfpm.WithDefaults(&p.Info)
	return p.Info.Validate()
}

func rewriteFileName(name string, filesMap StringMap) (string, error) {
	for k, v := range filesMap {
		if strings.HasPrefix(name, k) {
			return v + name[len(k):], nil
		}
	}
	return "", fmt.Errorf("can't rewrite %s", name)
}

func (p *Packager) AddFiles(fileS StringSlice) error {
	for _, f := range fileS {
		fileRemap := strings.Split(f, "=")
		if len(fileRemap) > 2 {
			return fmt.Errorf("filemap is invalid: %s\n", f)
		}

		var v string
		if len(fileRemap) > 1 {
			v = fileRemap[1]
		} else {
			v = "/" + fileRemap[0]
		}
		if _, ok := p.FilesMap[v]; ok {
			return fmt.Errorf("filemap produce duplicate: %s\n", v)
		}
		c := &files.Content{Source: fileRemap[0], Destination: v, Type: defaultStr}
		p.Info.Contents = append(p.Info.Contents, c)
		p.FilesMap[v] = c
	}
	return nil
}

func (p *Packager) AddSymlinks(fileS StringSlice) error {
	for _, f := range fileS {
		fileRemap := strings.Split(f, "=")
		if len(fileRemap) != 2 {
			return fmt.Errorf("symlink is invalid: %s\n", f)
		}
		if _, ok := p.FilesMap[fileRemap[1]]; ok {
			return fmt.Errorf("symlink try to overwrite existing: %s\n", fileRemap[1])
		}
		c := &files.Content{Source: fileRemap[0], Destination: fileRemap[1], Type: symlinkStr}
		p.Info.Contents = append(p.Info.Contents, c)
		p.FilesMap[fileRemap[1]] = c
	}
	return nil
}

func (p *Packager) setFiles(filesSet StringSlice, typ string) error {
	for _, f := range filesSet {
		var dpath string
		if strings.HasSuffix(f, "/") {
			dpath = f
		} else {
			dpath = f + "/"
		}
		for k, v := range p.FilesMap {
			if k == f || strings.HasPrefix(k, dpath) {
				if v.Type == "" {
					v.Type = typ
				}
			}
		}
	}
	return nil
}

func (p *Packager) SetConfigFiles(filesSet StringSlice) error {
	return p.setFiles(filesSet, configStr)
}

func (p *Packager) SetDocFiles(filesSet StringSlice) error {
	return p.setFiles(filesSet, docStr)
}

func (p *Packager) formatOutName(packager nfpm.Packager) string {
	s := p.OutName
	if s == "" {
		return packager.ConventionalFileName(&p.Info)
	}

	s = strings.ReplaceAll(s, "NAME", p.Info.Name)
	s = strings.ReplaceAll(s, "VERSION", p.Info.Version)
	s = strings.ReplaceAll(s, "ITERATION", p.Info.Release)
	s = strings.ReplaceAll(s, "ARCH", p.Info.Arch)
	s = strings.ReplaceAll(s, "PLATFORM", p.Info.Platform)

	return s
}

func (p *Packager) Do() (string, error) {
	packager, err := nfpm.Get(p.OutputType.String())
	if err != nil {
		return "", err
	}

	if len(p.PostUpgrade) > 0 {
		p.Info.RPM.Scripts.PostTrans = p.PostUpgrade
		p.Info.APK.Scripts.PostUpgrade = p.PostUpgrade
	}
	if len(p.PreUpgrade) > 0 {
		p.Info.RPM.Scripts.PreTrans = p.PostUpgrade
		p.Info.APK.Scripts.PreUpgrade = p.PostUpgrade
	}

	outName := p.formatOutName(packager)
	if p.OutDir == "" {
		// if no target was specified create a package in
		// current directory with a conventional file name
		p.Info.Target = outName
	} else {
		p.Info.Target = path.Join(p.OutDir, outName)
	}

	f, err := os.Create(p.Info.Target)
	if err != nil {
		return p.Info.Target, err
	}

	err = packager.Package(&p.Info, f)
	if err != nil {
		f.Close()
		os.Remove(p.Info.Target)
		return p.Info.Target, err
	}

	err = f.Close()
	if err != nil {
		os.Remove(p.Info.Target)
	}

	return p.Info.Target, nil
}