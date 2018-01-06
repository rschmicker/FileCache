package cache

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type CacheTable struct {
	Table       []CacheObject
	RamDiskPath string
	Size        uint64
	CurrentSize uint64
	Files       []string
	lock        sync.Mutex
}

type CacheObject struct {
	FilePath  string
	InProcess bool
	Completed bool
}

func (ct *CacheTable) Initialize() {
	os.RemoveAll(ct.RamDiskPath)
	err := os.Mkdir(ct.RamDiskPath, 0777)
	if err != nil {
		log.Fatal(err)
	}
	ct.CurrentSize = 0
	RemoveDuplicates(&ct.Files)
	if len(ct.Files) == 0 {
		log.Fatal("No Files found")
	}
	ct.Size = ct.AvailableRamSpace() / 2
	ct.Populate()
}

func (ct *CacheTable) Populate() {
	ct.lock.Lock()
	defer ct.lock.Unlock()
	end := len(ct.Files)
	for i := 0; i < len(ct.Files); i++ {
		if (ct.CurrentSize + GetFileSize(ct.Files[i])) > ct.Size {
			end = i
			break
		}
		_, name := filepath.Split(ct.Files[i])
		err := copyFileContents(ct.Files[i], ct.RamDiskPath+name)
		if err != nil {
			log.Fatal(err)
		}
		ct.CurrentSize += GetFileSize(ct.Files[i])
		co := CacheObject{ct.RamDiskPath + name, false, false}
		ct.Table = append(ct.Table, co)
		log.Println("Caching: " + name)
	}
	ct.Files = ct.Files[end:]
}

func (ct *CacheTable) AvailableRamSpace() uint64 {
	var stat syscall.Statfs_t
	syscall.Statfs(ct.RamDiskPath, &stat)
	return (stat.Bavail * uint64(stat.Bsize))
}

func GetFileSize(path string) uint64 {
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	return uint64(fi.Size())
}

func (ct *CacheTable) Runner() {
	for {
		if ct.IsEmpty() {
			break
		}
		ct.GarbageCollector()
		ct.Populate()
	}
}

func (ct *CacheTable) GarbageCollector() {
	ct.lock.Lock()
	defer ct.lock.Unlock()
	for i := 0; i < len(ct.Table); i++ {
		file := ct.Table[i]
		if file.Completed {
			fileSize := GetFileSize(file.FilePath)
			err := os.Remove(file.FilePath)
			if err != nil {
				log.Fatal(err)
			}
			ct.Table[i] = ct.Table[len(ct.Table)-1]
			ct.Table = ct.Table[:len(ct.Table)-1]
			_, name := filepath.Split(file.FilePath)
			log.Println("Removed: " + name + " from cache")
			ct.CurrentSize -= fileSize
		}
	}
}

func (ct *CacheTable) Completed(path string) {
	if path == "" {
		return
	}
	_, name := filepath.Split(path)
	for i := 0; i < len(ct.Table); i++ {
		_, ramName := filepath.Split(ct.Table[i].FilePath)
		if ramName == name {
			ct.Table[i].Completed = true
		}
	}
}

func (ct *CacheTable) IsEmpty() bool {
	ct.lock.Lock()
	defer ct.lock.Unlock()
	return (len(ct.Table) == 0 && len(ct.Files) == 0)
}

func (ct *CacheTable) GetFilePath() string {
	path := ""
	ct.lock.Lock()
	defer ct.lock.Unlock()
	for i := 0; i < len(ct.Table); i++ {
		if !ct.Table[i].InProcess {
			path = ct.Table[i].FilePath
			ct.Table[i].InProcess = true
			ct.RemoveFileFromList(ct.Table[i].FilePath)
			break
		}
	}
	return path
}

func (ct *CacheTable) RemoveFileFromList(filePath string) {
	_, name := filepath.Split(filePath)
	for i, file := range ct.Files {
		if strings.Contains(file, name) {
			ct.Files[i] = ct.Files[len(ct.Files)-1]
			ct.Files = ct.Files[:len(ct.Files)-1]
		}
	}
}

func (ct *CacheTable) Close() {
	err := os.RemoveAll(ct.RamDiskPath)
	if err != nil {
		log.Fatal(err)
	}
}

func RemoveDuplicates(xs *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x] {
			found[x] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		log.Fatal(err)
	}
	err = out.Sync()
	return
}
