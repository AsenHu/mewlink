package kv

import (
	"encoding/gob"
	"os"
	"sync"

	"maunium.net/go/mautrix/id"
)

type KVStore struct {
	path     string
	mu       sync.RWMutex
	forward  map[int64]id.RoomID
	backward map[id.RoomID]int64
}

func NewKVStore(path string) *KVStore {
	return &KVStore{
		path:     path,
		forward:  make(map[int64]id.RoomID),
		backward: make(map[id.RoomID]int64),
	}
}

func (kv *KVStore) Set(chatID int64, roomID id.RoomID) error {
	kv.mu.Lock()
	kv.forward[chatID] = roomID
	kv.backward[roomID] = chatID
	kv.mu.Unlock()
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	f, err := os.Create(kv.path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return enc.Encode(kv.forward)
}

func (kv *KVStore) GetChatID(key id.RoomID) (int64, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok := kv.backward[key]
	return val, ok
}

func (kv *KVStore) GetRoomID(key int64) (id.RoomID, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok := kv.forward[key]
	return val, ok
}

func (kv *KVStore) Load(filename string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	err = dec.Decode(&kv.forward)
	if err != nil {
		return err
	}
	for k, v := range kv.forward {
		kv.backward[v] = k
	}
	return nil
}
