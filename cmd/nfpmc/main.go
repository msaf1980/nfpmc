package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	flag "github.com/spf13/pflag"
)

// func filePathWalkDir(root string) ([]string, error) {
// 	var files []string
// 	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
// 		if info == nil {
// 			return fmt.Errorf("%s not a dir or file", path)
// 		}
// 		if !info.IsDir() {
// 			files = append(files, path)
// 		}
// 		return nil
// 	})
// 	return files, err
// }

func main() {
	var (
		p   Packager
		dir string

		err error

		configFiles  StringSlice
		docFiles     StringSlice
		symlinkFiles StringSlice

		noDebSystemdRestart bool

		overwrite bool
	)

	flag.CommandLine.SortFlags = false

	flag.VarP(&p.InputType, "input-type", "s", "the package type to use as input (dir)")
	flag.StringVarP(&dir, "chdir", "C", "", "(OPTIONAL) directory for searching files (not scripts)")

	flag.VarP(&p.OutputType, "output-type", "t", "the type of package you want to create (rpm deb apk)")
	flag.StringVar(&p.OutDir, "target", "", "(OPTIONAL) dir for store package")

	flag.BoolVarP(&overwrite, "force", "f", false, "Force output even if it will overwrite an existing file")
	flag.StringVarP(&p.OutName, "package", "p", "NAME-VERSION-ITERATION.ARCH.rpm", "The package file path to output (use NAME, VERSION, ITERATION and ARCH for substitution).")
	flag.StringVarP(&p.Info.Name, "name", "n", "", "The name to give to the package")
	flag.StringVarP(&p.Info.Version, "version", "v", "", "The version to give to the package")
	flag.StringVarP(&p.Info.Release, "iteration", "i", "1", "The iteration to give to the package. RPM calls this the 'release'")
	flag.StringVarP(&p.Info.Arch, "architecture", "a", "", "The architecture name. Usually matches 'uname -m'")
	flag.StringVarP(&p.Info.Platform, "platform", "P", "", "The platform name.")
	flag.StringVarP(&p.Info.License, "license", "l", "", "(optional) license name for this package")
	flag.StringVarP(&p.Info.Maintainer, "maintainer", "m", "", "The maintainer of this package. (default: <msv@power.test.int>")
	flag.StringVarP(&p.Info.Description, "description", "d", "no description", "Add a description for this package")
	flag.StringVarP(&p.Info.Homepage, "url", "u", "", "(optional) Homepage for this package")
	flag.StringVar(&p.Info.Section, "--category", "none", "category this package belongs to")

	flag.BoolVar(&noDebSystemdRestart, "no-deb-systemd-restart-after-upgrade", false, "(FAKE) fpm compability parameter, ignored")

	flag.Var(&configFiles, "config-files", "Mark a file in the package as being a config file. This uses 'conffiles' in debs and %config in rpm. If you have multiple files to mark as configuration files, specify this flag multiple times. If argument is directory all files inside it will be recursively marked as config files.")
	flag.Var(&docFiles, "doc-files", "Mark a file in the package as being a doc file.")
	flag.Var(&symlinkFiles, "symlink-files", "Create symlink.")

	flag.StringVar(&p.Info.RPM.Compression, "rpm-compression", "gzip", "Compression method. gzip works on the most platform [none|xz|xzmt|gzip|bzip2].")
	flag.StringVar(&p.Info.Platform, "rpm-os", "linux", "Compression method. gzip works on the most platform [none|xz|xzmt|gzip|bzip2].")

	flag.StringVar(&p.Info.Scripts.PostInstall, "post-install", "", "(DEPRECATED) use --after-install")
	flag.StringVar(&p.Info.Scripts.PreInstall, "pre-install", "", "(DEPRECATED) use --before-install")
	flag.StringVar(&p.Info.Scripts.PostRemove, "post-uninstall", "", "(DEPRECATED) use --after-remove")
	flag.StringVar(&p.Info.Scripts.PreRemove, "pre-uninstall", "", "(DEPRECATED) use --before-remove")
	flag.StringVar(&p.Info.Scripts.PostInstall, "after-install", "", "A script to be run after package installation")
	flag.StringVar(&p.Info.Scripts.PreInstall, "before-install", "", "A script to be run before package installation")
	flag.StringVar(&p.Info.Scripts.PostRemove, "after-remove", "", "A script to be run after package removal")
	flag.StringVar(&p.Info.Scripts.PreRemove, "before-remove", "", "A script to be run before package removal")

	flag.StringVar(&p.PreUpgrade, "after-upgrade", "", "A script to be run after package upgrade (only for rpm, apk)")
	flag.StringVar(&p.PostUpgrade, "before-upgrade", "", "A script to be run before package upgrade (only for rpm, apk)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Use: %s FILE1[=DEST1] [ [FILE2[=DEST2] ..]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if err = p.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	err = p.AddFiles(flag.CommandLine.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	err = p.AddSymlinks(symlinkFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	if len(configFiles) == 0 {
		for _, f := range p.FilesMap {
			if strings.HasPrefix(f.Destination, "/etc") {
				configFiles = append(configFiles, "/etc")
				break
			}
		}
	}
	err = p.SetConfigFiles(configFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	if len(docFiles) == 0 {
		for _, f := range p.FilesMap {
			if strings.HasPrefix(f.Destination, "/usr/share") {
				docFiles = append(docFiles, "/usr/share")
				break
			}
		}
	}
	err = p.SetDocFiles(docFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	if p.Info.Contents.Len() == 0 {
		fmt.Fprintf(os.Stderr, "filemap is empty\n")
		os.Exit(1)
	}

	// append dir
	if len(dir) > 0 || dir == "." {
		for _, f := range p.FilesMap {
			if f.Source[0] != '/' {
				f.Source = path.Join(dir, f.Source)
			}
		}
	}

	err = p.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	packageName, err := p.Do(overwrite)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("created package: %s\n", packageName)
}
