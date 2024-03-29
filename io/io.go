package io

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/shoaib42/remote-move/conf"
)

type IOHelpers interface {
	DoMvChown(from, what, where string) error
	DoCpChown(from, what, where string) error
	GetDestDirList() ([]string, error)
	GetSrcMapItems() (map[string][]string, error)
}

type IoConf struct {
	mu           sync.Mutex
	destRootDir  string
	srcDirs      []string
	excludeDirs  map[string]conf.VoidT
	srcDirsItems map[string][]string
	uid          int
	gid          int
}

func NewIOHelper(srcDirs []string, destDirRoot string, excludeDirs []string, uid, gid int) (IOHelpers, error) {

	srcDirsSorted := make([]string, len(srcDirs))
	copy(srcDirsSorted, srcDirs)
	sort.Strings(srcDirsSorted)
	excldDirs := make(map[string]conf.VoidT, 0)

	for _, e := range excludeDirs {
		excldDirs[e] = conf.Void
	}

	return &IoConf{
		destRootDir:  destDirRoot,
		srcDirs:      srcDirsSorted,
		excludeDirs:  excldDirs,
		srcDirsItems: nil,
		uid:          uid,
		gid:          gid,
	}, nil
}

func find(list []string, what string) bool {
	_, found := sort.Find(len(list), func(i int) int {
		return strings.Compare(what, list[i])
	})
	return found
}

func checkIfExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsExist(err)
}

func (i *IoConf) doChown(dest string) error {
	return filepath.Walk(dest, func(name string, info os.FileInfo, err error) error {
		if nil == err {
			err = os.Chown(name, i.uid, i.gid)
		}
		return err
	})
}

func (i *IoConf) checkCopyOrMoveValid(from, what, where string) error {
	dest := i.destRootDir + "/" + where
	if from == "" {
		return errors.New("source directory was not provided")
	}
	if what == "" {
		return errors.New("file/dir to move|copy was not provided")
	}
	if where == "" {
		return errors.New("destination directory was not provided")
	}
	if from == dest {
		return errors.New("src and dest directories cannot be the same")
	}
	srcMapList, err := i.GetSrcMapItems()
	if nil != err {
		return err
	}

	items, ok := srcMapList[from]
	if !ok {
		return errors.New("source directory not accessible")
	}

	if !find(items, what) {
		return errors.New("item not found in source directory")
	}

	destDirs, err := i.GetDestDirList()
	if !find(destDirs, where) {
		return errors.New("destination directory not accessible")
	}
	return nil
}

// Start https://stackoverflow.com/questions/51779243/copy-a-folder-in-go
func copyDir(src, dest string) error {

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// copy to this path
		outpath := filepath.Join(dest, strings.TrimPrefix(path, src))

		if info.IsDir() {
			os.MkdirAll(outpath, info.Mode())
			return nil // means recursive
		}

		// skip irregular file
		if !info.Mode().IsRegular() {
			return nil
		}

		// copy contents of regular file efficiently

		// open input
		in, _ := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		// create output
		fh, err := os.Create(outpath)
		if err != nil {
			return err
		}
		defer fh.Close()

		// make it the same
		fh.Chmod(info.Mode())

		// copy content
		_, err = io.Copy(fh, in)
		return err
	})
}

// End https://stackoverflow.com/questions/51779243/copy-a-folder-in-go

func (i *IoConf) doCpChown(from, what, where string) error {
	if err := i.checkCopyOrMoveValid(from, what, where); err != nil {
		return err
	}

	src := from + "/" + what
	dest := i.destRootDir + "/" + where + "/" + what

	if err := copyDir(src, dest); nil != err {
		return err
	}

	return i.doChown(dest)
}

func (i *IoConf) DoCpChown(from, what, where string) error {
	i.mu.Lock()
	err := i.doCpChown(from, what, where)
	i.mu.Unlock()
	return err
}

func (i *IoConf) doMvChown(from, what, where string) error {
	if err := i.checkCopyOrMoveValid(from, what, where); err != nil {
		return err
	}

	src := from + "/" + what
	dest := i.destRootDir + "/" + where + "/" + what

	if err := os.Rename(src, dest); nil != err {
		return err
	}

	return i.doChown(dest)
}

func (i *IoConf) DoMvChown(from, what, where string) error {
	i.mu.Lock()
	err := i.doMvChown(from, what, where)
	i.mu.Unlock()
	return err
}

func (i *IoConf) GetSrcMapItems() (map[string][]string, error) {
	ret := make(map[string][]string, 0)
	var err error
	for _, s := range i.srcDirs {
		list := make([]string, 0)
		entries, err := os.ReadDir(s)
		if nil != err {
			return ret, err
		}
		for _, e := range entries {
			notstickit := false
			if e.IsDir() {
				_, notstickit = i.excludeDirs[e.Name()]
			}
			if !notstickit {
				list = append(list, e.Name())
			}
		}
		sort.Strings(list)
		ret[s] = list
	}
	return ret, err
}

func (i *IoConf) GetDestDirList() ([]string, error) {
	ret := make([]string, 0)
	entries, err := os.ReadDir(i.destRootDir)
	if nil == err {
		for _, e := range entries {
			if e.IsDir() {
				if _, ok := i.excludeDirs[e.Name()]; !ok {
					ret = append(ret, e.Name())
				}
			}
		}
	}
	sort.Strings(ret)
	return ret, err
}
