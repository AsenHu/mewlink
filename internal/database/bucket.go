package database

import (
	"sync"

	"go.etcd.io/bbolt"
)

// 给每个 bucket 分配一个 uint8 的 id

const (
	bucketEventListEvents     uint8 = 0
	bucketRoomListRoomInfo    uint8 = 1
	bucketRoomListChatIDIndex uint8 = 2
	bucketRoomListRoomIDIndex uint8 = 3
)

// 对于每一个 bucket，都应该有一个对应的结构体

type Bucket struct {
	database   *bbolt.DB
	bucket     []byte
	keyLen     int
	IndexMutex sync.Map
}

// Bucket 应该可以读改删

// 读取

func (b *Bucket) Get(key []byte) (value []byte, err error) {
	err = b.database.View(func(tx *bbolt.Tx) error {
		value = tx.Bucket(b.bucket).Get(key)
		return nil
	})
	return
}

// 写入/更新

func (b *Bucket) Put(key, value []byte) (err error) {
	err = b.database.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(b.bucket).Put(key, value)
	})
	return
}

// 删除

func (b *Bucket) Delete(key []byte) (err error) {
	err = b.database.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(b.bucket).Delete(key)
	})
	return
}

// Bucket 应该能够检查自己是否存在

func (b *Bucket) Exists() (exists bool, err error) {
	err = b.database.View(func(tx *bbolt.Tx) error {
		if tx.Bucket(b.bucket) == nil {
			exists = false
		} else {
			exists = true
		}
		return nil
	})
	return
}

// Bucket 应该能够创建自己

func (b *Bucket) Create() (err error) {
	err = b.database.Update(func(tx *bbolt.Tx) (err error) {
		_, err = tx.CreateBucket(b.bucket)
		return
	})
	return
}

// Bucket 应该能够删除自己

func (b *Bucket) DeleteBucket() (err error) {
	err = b.database.Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket(b.bucket)
	})
	return
}

// 在 Bucket 里寻找一个没有被使用的 key
// 应该在初始化的时候将 keyLen 设置为 1
// 返回 err 的时候一定要释放 lock

func (b *Bucket) FindUnusedKey() (key []byte, lock *sync.RWMutex, err error) {
	// 生成一个随机 1 字节的 key
	key = make([]byte, b.keyLen)
	_, err = Rand.Read(key)
	if err != nil {
		return
	}

	var v []byte
	for {
		// 锁定 key
		lockByte, _ := b.IndexMutex.LoadOrStore(string(key), &sync.RWMutex{})
		lock = lockByte.(*sync.RWMutex)
		lock.Lock()

		// 检查 key 是否已经被使用
		v, err = b.Get(key)
		if err != nil {
			lock.Unlock()
			return
		}
		if v == nil {
			break
		}

		// key 已经被使用，释放锁，keyLen + 1
		lock.Unlock()
		// 这里 keyLen 不是线程安全的，但是如果 keyLen 意外的没被修改，也不会有问题，所以不加锁
		b.keyLen += 1

		// 生成新的 key
		key = make([]byte, b.keyLen)
		_, err = Rand.Read(key)
		if err != nil {
			lock.Unlock()
			return
		}
	}

	// 这里返回了 key 和 lock，上层函数处理完 value 后应该释放 lock
	return
}
