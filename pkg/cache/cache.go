package cache

import (
	"container/list"
	"io/ioutil"
	"os"
	"path/filepath"
)

type LRUCache struct {
	maxFiles    int
	cacheList   *list.List
	cacheMap    map[string]*list.Element
	cacheFolder string
}

type CacheFile struct {
	Path string
	Data []byte
}

func New(maxFiles int, cacheFolder string) (*LRUCache, error) {
	// Create cache folder if it doesn't exist
	err := os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		return nil, err
	}

	cache := &LRUCache{
		maxFiles:    maxFiles,
		cacheList:   list.New(),
		cacheMap:    make(map[string]*list.Element),
		cacheFolder: cacheFolder,
	}

	return cache, nil
}

func (c *LRUCache) GetFile(path string) ([]byte, error) {
	// Check if the file is in the cache
	if element, ok := c.cacheMap[path]; ok {
		// Move the file to the front of the cache list (MRU position)
		c.cacheList.MoveToFront(element)
		return element.Value.(*CacheFile).Data, nil
	}

	// If the file is not in the cache, load it from the disk
	filePath := filepath.Join(c.cacheFolder, path)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Create a new CacheFile object and add it to the cache
	cacheFile := &CacheFile{
		Path: path,
		Data: data,
	}
	element := c.cacheList.PushFront(cacheFile)
	c.cacheMap[path] = element

	// If the cache size exceeds the maximum limit, remove the least recently used file
	if c.cacheList.Len() > c.maxFiles {
		c.removeLRUFile()
	}

	return data, nil
}

func (c *LRUCache) PutFile(path string, data []byte) error {
	filePath := filepath.Join(c.cacheFolder, path)
	err := ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	// If the file already exists in the cache, update its data and move it to the front
	if element, ok := c.cacheMap[path]; ok {
		element.Value.(*CacheFile).Data = data
		c.cacheList.MoveToFront(element)
		return nil
	}

	// Create a new CacheFile object and add it to the cache
	cacheFile := &CacheFile{
		Path: path,
		Data: data,
	}
	element := c.cacheList.PushFront(cacheFile)
	c.cacheMap[path] = element

	// If the cache size exceeds the maximum limit, remove the least recently used file
	if c.cacheList.Len() > c.maxFiles {
		c.removeLRUFile()
	}

	return nil
}

func (c *LRUCache) removeLRUFile() {
	// Get the least recently used file from the back of the cache list
	element := c.cacheList.Back()
	if element == nil {
		return
	}

	// Remove the file from the cache list and the cache map
	c.cacheList.Remove(element)
	delete(c.cacheMap, element.Value.(*CacheFile).Path)

	// Remove the file from the disk
	filePath := filepath.Join(c.cacheFolder, element.Value.(*CacheFile).Path)
	_ = os.Remove(filePath)
}

func (c *LRUCache) ClearCache() error {
	// Clear the cache list and the cache map
	c.cacheList.Init()
	c.cacheMap = make(map[string]*list.Element)

	// Remove all files from the cache folder
	return os.RemoveAll(c.cacheFolder)
}
