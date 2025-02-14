package database

import (
	"go.etcd.io/bbolt"
	"maunium.net/go/mautrix/id"
)

type EventList struct {
	events Bucket
}

func newEventList(db *bbolt.DB) (el *EventList, err error) {
	el = &EventList{
		events: Bucket{
			database: db,
			bucket:   []byte{bucketEventListEvents},
			keyLen:   1,
		},
	}

	// 检查 bucket 是否存在
	exi, err := el.events.Exists()
	// 如果不存在则创建
	if !exi {
		err = el.events.Create()
	}
	return
}

func (el *EventList) IsExi(id id.EventID) (exi bool, err error) {
	value, err := el.events.Get([]byte(id.String()))
	if err != nil {
		return
	}
	if value != nil {
		exi = true
	}
	return
}

func (el *EventList) Set(id id.EventID) (err error) {
	err = el.events.Put([]byte(id.String()), []byte{})
	return
}
