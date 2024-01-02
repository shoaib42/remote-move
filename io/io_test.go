package io

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shoaib42/remote-move/conf"
)

var testRootDir = "test"
var srcDirs = []string{testRootDir + "/src1", testRootDir + "/src2", testRootDir + "/src3"}
var destRootDir = testRootDir + "/dest"
var destDirs = []string{destRootDir + "/land1", destRootDir + "/land2", destRootDir + "/land3"}
var dirWantExclude = []string{"exclude1", "exclude2"}
var filesCreate = []string{"file1", "file2"}
var dirsCreate = []string{"dir1", "dir2"}

func mock_data() *conf.Configuration {
	for _, p := range srcDirs {
		os.MkdirAll(p, os.ModePerm)
		for _, d := range dirsCreate {
			lvl1 := p + "/" + d
			os.MkdirAll(lvl1, os.ModePerm)
			for _, d2 := range dirsCreate {
				lvl2 := lvl1 + "/sub" + d2
				os.MkdirAll(lvl2, os.ModePerm)
				for _, f := range filesCreate {
					os.Create(lvl2 + "/" + f)
				}
			}
		}
		for _, f := range filesCreate {
			os.Create(p + "/" + f)
		}
		for _, e := range dirWantExclude {
			os.MkdirAll(p+"/"+e, os.ModePerm)
		}
	}

	for _, p := range destDirs {
		os.MkdirAll(p, os.ModePerm)
	}

	return &conf.Configuration{
		SrcDirs:     srcDirs,
		DestRootDir: destRootDir,
		ExcludeDirs: dirWantExclude,
		Uid:         os.Geteuid(),
		Gid:         os.Getegid(),
	}
}

func tearDown() {
	os.RemoveAll(testRootDir)
}

func TestGetSrcs(t *testing.T) {
	mock_data()
	tearDown()
}

func TestCreateIOHelper(t *testing.T) {
	conf := mock_data()
	defer tearDown()
	_, err := NewIOHelper(conf.SrcDirs, conf.DestRootDir, conf.ExcludeDirs, conf.Uid, conf.Gid)
	if nil != err {
		t.Fatalf("Could not create io helper")
	}
}

func TestGetDestDir(t *testing.T) {
	conf := mock_data()
	defer tearDown()
	ioh, err := NewIOHelper(conf.SrcDirs, conf.DestRootDir, conf.ExcludeDirs, conf.Uid, conf.Gid)
	if nil != err {
		t.Fatalf("Could not create io helper")
	}
	dest, err := ioh.GetDestDirList()

	if nil != err {
		t.Fatalf("Could not get dest directories")
	}
	if len(destDirs) != len(dest) {
		t.Fatalf("the number of destination dirs is incorrect")
	}
	if !sort.StringsAreSorted(dest) {
		t.Fatalf("dest dir list is not sorted")
	}
	for _, d := range destDirs {
		if _, found := sort.Find(len(dest), func(i int) int {
			return strings.Compare(d, destRootDir+"/"+dest[i])
		}); !found {
			t.Fatalf("valid dest dir was not found %s %v", d, dest)
		}
	}
	for _, d := range dirWantExclude {
		if _, found := sort.Find(len(dest), func(i int) int {
			return strings.Compare(d, dest[i])
		}); found {
			t.Fatalf("dir %s should be excluded %v", d, dest)
		}
	}

}

func TestGetSourceItems(t *testing.T) {
	conf := mock_data()
	defer tearDown()
	ioh, err := NewIOHelper(conf.SrcDirs, conf.DestRootDir, conf.ExcludeDirs, conf.Uid, conf.Gid)
	if nil != err {
		t.Fatalf("Could not create io helper")
	}

	mup, err := ioh.GetSrcMapItems()
	if nil != err {
		t.Fatalf("Could not get Map of arrays")
	}

	if len(mup) != len(srcDirs) {
		t.Fatalf("The length of map is incorrect")
	}

	for _, s := range srcDirs {
		items, ok := mup[s]
		if !ok {
			t.Fatalf("map does not contain the key %s", s)
		}
		if !sort.StringsAreSorted(items) {
			t.Fatalf("listing of dir : %s, is not sorted %v", s, items)
		}
		for _, f := range filesCreate {
			if _, found := sort.Find(len(items), func(i int) int {
				return strings.Compare(f, items[i])
			}); !found {
				t.Fatalf("file %s should be available %v", f, items)
			}
		}
		for _, d := range dirsCreate {
			if _, found := sort.Find(len(items), func(i int) int {
				return strings.Compare(d, items[i])
			}); !found {
				t.Fatalf("dir %s should be available %v", d, items)
			}
		}
		for _, d := range dirWantExclude {
			if _, found := sort.Find(len(items), func(i int) int {
				return strings.Compare(d, items[i])
			}); found {
				t.Fatalf("dir %s should be excluded %v", d, items)
			}
		}
	}
}

func TestMove(t *testing.T) {
	conf := mock_data()
	defer tearDown()
	ioh, err := NewIOHelper(conf.SrcDirs, conf.DestRootDir, conf.ExcludeDirs, conf.Uid, conf.Gid)
	if nil != err {
		t.Fatalf("Could not create io helper")
	}
	filetomove := "filetomove"
	fileSrcFullPath := srcDirs[0] + "/" + filetomove
	fileDestFullPath := destDirs[0] + "/" + filetomove
	//add a file to move
	os.Create(fileSrcFullPath)

	dest := strings.Replace(destDirs[0], destRootDir+"/", "", 1)
	err = ioh.DoMvChown(srcDirs[0], filetomove, dest)
	if nil != err {
		t.Fatalf("Failed to move with error %v", err)
	}

	//check to see if we have the file in the dest and is not in the source
	_, err = os.Stat(fileDestFullPath)
	if os.IsNotExist(err) {
		t.Fatalf("File not found in destination %s, while checking %s", dest, fileDestFullPath)
	}

	//uid gid not tested!

	_, err = os.Stat(fileSrcFullPath)
	if os.IsExist(err) {
		t.Fatalf("File found in source %s, while checking %s", srcDirs[0], fileSrcFullPath)
	}

	mup, err := ioh.GetSrcMapItems()
	if nil != err {
		t.Fatalf("Failed to get src map after move")
	}
	if _, found := sort.Find(len(mup[srcDirs[0]]), func(i int) int {
		return strings.Compare(filetomove, mup[srcDirs[0]][i])
	}); found {
		t.Fatalf("dir %s should not contain %s", mup[srcDirs[0]], filetomove)
	}

}

func TestCopy(t *testing.T) {
	conf := mock_data()
	defer tearDown()
	ioh, err := NewIOHelper(conf.SrcDirs, conf.DestRootDir, conf.ExcludeDirs, conf.Uid, conf.Gid)
	if nil != err {
		t.Fatalf("Could not create io helper")
	}

	fileSrcFullPath := srcDirs[0] + "/" + dirsCreate[0]
	fileDestFullPath := destDirs[0] + "/" + dirsCreate[0]
	//add a file to copy
	os.Create(fileSrcFullPath + "/" + "extraFile")

	dest := strings.Replace(destDirs[0], destRootDir+"/", "", 1)
	err = ioh.DoCpChown(srcDirs[0], dirsCreate[0], dest)
	if nil != err {
		t.Fatalf("Failed to move with error %v", err)
	}

	//check to see if we have the file in the dest and is not in the source
	_, err = os.Stat(fileDestFullPath)
	if os.IsNotExist(err) {
		t.Fatalf("File not found in destination %s, while checking %s", dest, fileDestFullPath)
	}

	//uid gid not tested!

	_, err = os.Stat(fileSrcFullPath)
	if os.IsNotExist(err) {
		t.Fatalf("File not found in source %s, while checking %s", srcDirs[0], fileSrcFullPath)
	}

	// check if all files copied
	err = filepath.Walk(fileSrcFullPath, func(name string, info os.FileInfo, err error) error {
		if nil == err {
			destloc := filepath.Join(fileDestFullPath, strings.TrimPrefix(name, fileSrcFullPath))
			_, err = os.Stat(destloc)
			if os.IsNotExist(err) {
				return err
			}
		}
		return nil
	})

	if nil != err {
		t.Fatalf("Failed comparing source and desitination files")
	}

	//check source dir if all files exists as before
	//should be same as that of dir2, dir2 will be source of truth since dir1 has "extraFile"
	checkAgainst := srcDirs[1] + "/" + dirsCreate[0]
	err = filepath.Walk(checkAgainst, func(name string, info os.FileInfo, err error) error {
		if nil == err {
			destloc := filepath.Join(fileSrcFullPath, strings.TrimPrefix(name, checkAgainst))
			_, err = os.Stat(destloc)
			if os.IsNotExist(err) {
				return err
			}
		}
		return nil
	})

	if nil != err {
		t.Fatalf("Failed checking integrity of source file")
	}

}
