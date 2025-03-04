package database

import (
	"go.etcd.io/bbolt"
)

type DataBase struct {
	database  *bbolt.DB
	RoomList  *RoomList
	EventList *EventList
}

func NewDataBase(path string) (db *DataBase, err error) {
	database, err := bbolt.Open(path, 0600, bbolt.DefaultOptions)
	if err != nil {
		return
	}

	db = &DataBase{
		database: database,
	}

	db.RoomList, err = newRoomList(database)
	if err != nil {
		return
	}

	db.EventList, err = newEventList(database)
	if err != nil {
		return
	}

	return
}

func (db *DataBase) Close() error {
	return db.database.Close()
}
