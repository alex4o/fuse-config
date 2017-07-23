package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"
)
import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)
import "github.com/spf13/viper"

// JsonFS ...
type JsonFS struct {
	pathfs.FileSystem
}

// JsonFile ...
type JsonFile struct {
	field string
	nodefs.File
}

func (file *JsonFile) Utimens(atime *time.Time, mtime *time.Time) fuse.Status {
	return fuse.OK
}

func (file *JsonFile) SetInode(n *nodefs.Inode) {

}

func (file *JsonFile) Flush() fuse.Status {
	// todo: check write ot file
	return fuse.OK
}

func (file *JsonFile) Release() {
	err := viper.MergeInConfig()
	if err != nil {
		log.Println("Confing merging faild.")

	}

	log.Println("File released")
}

func (file *JsonFile) String() string {
	return fmt.Sprintf("dataFile(%x)", file.field)
}

func (file *JsonFile) GetAttr(out *fuse.Attr) fuse.Status {
	log.Printf("GetAttr: %s %v", file.field, viper.GetString(file.field))

	out.Mode = fuse.S_IFREG | 0777
	out.Size = uint64(len(viper.GetString(file.field)))
	return fuse.OK
}

func (file *JsonFile) Truncate(size uint64) fuse.Status {
	return fuse.OK
}

// Read ...
func (file *JsonFile) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	log.Printf("Read: %s %v", file.field, viper.GetString(file.field))

	return fuse.ReadResultData([]byte(viper.GetString(file.field))), fuse.OK
}

// Write ...
func (file *JsonFile) Write(dest []byte, off int64) (written uint32, code fuse.Status) {
	log.Printf("Write: %s %v", file.field, dest)

	viper.Set(file.field, dest)

	return uint32(len(dest)), fuse.OK
}

// GetAttr ...
func (fs *JsonFS) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {

	if name == "" {
		return &fuse.Attr{
			Mode: fuse.S_IFDIR | 0755,
		}, fuse.OK
	}

	name = strings.Replace(name, "/", ".", -1)

	if !viper.IsSet(name) {
		log.Printf("Not in config: %s", name)
		return nil, fuse.ENOENT
	}

	value := viper.Get(name)

	//log.Printf("name: %s %v", name, value)

	_, ok := value.(map[string]interface{})

	if ok {
		return &fuse.Attr{
			Mode: fuse.S_IFDIR | 0755,
		}, fuse.OK
	} else {
		return &fuse.Attr{
			Mode: fuse.S_IFREG | 0777,
			Size: uint64(len(viper.GetString(name))),
		}, fuse.OK
	}

	// switch name {
	// case "file.txt":
	// 	return &fuse.Attr{
	// 		Mode: fuse.S_IFREG | 0644, Size: uint64(len(name)),
	// 	}, fuse.OK
	// case "":
	// 	return &fuse.Attr{
	// 		Mode: fuse.S_IFDIR | 0755,
	// 	}, fuse.OK
	// }
	// return &fuse.Attr{
	// 	Mode: fuse.S_IFREG | 0644,
	// }, fuse.OK
}

// OpenDir ...
func (fs *JsonFS) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	localViper := viper.GetViper()

	if name != "" {
		localViper = viper.Sub(name)

		if !viper.InConfig(name) {
			return nil, fuse.ENOENT
		}
	} else {

	}

	c = []fuse.DirEntry{}
	prev := ""

	for key, value := range localViper.AllSettings() {
		path := strings.Split(key, ".")
		depth := len(path)

		//value := localViper.Get(path[0])
		v, ok := value.(map[string]interface{})

		log.Printf("key: %s depth: %d %v ok: %v", key, depth, v, ok)

		if prev != path[0] {
			if depth > 1 || ok {
				c = append(c, fuse.DirEntry{Name: path[0], Mode: fuse.S_IFDIR})
			} else {
				c = append(c, fuse.DirEntry{Name: key, Mode: fuse.S_IFREG})
			}
		}

		prev = path[0]

	}

	return c, fuse.OK
}

// Mkdir
func (fs *JsonFS) Mkdir(name string, mode uint32, context *fuse.Context) fuse.Status {
	name = strings.Replace(name, "/", ".", -1)
	viper.Set(name, map[string]interface{}{})

	return fuse.OK
}

// Open ...
func (fs *JsonFS) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	if flags&fuse.O_ANYWRITE != 0 {
		//return nil, fuse.EPERM
	}

	f := new(JsonFile)
	f.field = strings.Replace(name, "/", ".", -1)
	return f, fuse.OK

	// return nodefs.NewDataFile([]byte("Hello: " + name)), fuse.OK
}

// Create ..
func (fs *JsonFS) Create(name string, flags uint32, mode uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	name = strings.Replace(name, "/", ".", -1)
	viper.Set(name, nil)
	f := new(JsonFile)
	f.field = name
	return f, fuse.OK

	// return nodefs.NewDataFile([]byte("Hello: " + name)), fuse.OK
}

// Access ...
func (fs *JsonFS) Access(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	log.Println(name)
	log.Println(mode)
	return fuse.OK
}

func main() {

	flag.Parse()
	if len(flag.Args()) < 2 {
		log.Fatal("Usage:\n  hello file(withouth ext) MOUNTPOINT")
	}

	viper.SetConfigName(flag.Arg(0))
	viper.AddConfigPath(".")
	viper.Debug()

	err := viper.ReadInConfig() // Find and read the config file

	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	nfs := pathfs.NewPathNodeFs(&JsonFS{FileSystem: pathfs.NewDefaultFileSystem()}, nil)

	server, _, err := nodefs.MountRoot(flag.Arg(1), nfs.Root(), nil)

	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	server.Serve()
}
