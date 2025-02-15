package database

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"maunium.net/go/mautrix/id"
)

type EventList struct {
	Mutex            sync.RWMutex
	Path             string
	IsSyncedWithFile bool
	ByEventID        map[id.EventID]EventInfo
}

type EventInfo struct {
	EventID id.EventID
}

func (el *EventList) GetEventInfoByEventID(eventID id.EventID) (EventInfo, bool) {
	el.Mutex.RLock()
	defer el.Mutex.RUnlock()
	val, ok := el.ByEventID[eventID]
	return val, ok
}

func (el *EventList) Load() error {
	f, err := os.Open(el.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var list []EventInfo
	err = dec.Decode(&list)
	if err != nil {
		return err
	}

	el.Mutex.Lock()
	defer el.Mutex.Unlock()
	for _, event := range list {
		el.ByEventID[event.EventID] = event
	}
	return nil
}

func (el *EventList) Set(info EventInfo) (err error) {
	el.Mutex.Lock()
	defer el.Mutex.Unlock()
	el.ByEventID[info.EventID] = info
	return nil
}

func (el *EventList) SaveNow() error {
	f, err := os.Create(el.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	var list []EventInfo
	for _, event := range el.ByEventID {
		list = append(list, event)
	}

	return enc.Encode(list)
}

func (el *EventList) LazySave(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("LazySave started in EventList")
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("LazySave cancelled in EventList")
				el.Mutex.Lock()
				err := el.SaveNow()
				el.Mutex.Unlock()
				if err != nil {
					log.Warn().Err(err).Msg("LazySave failed in EventList during context cancellation.")
				} else {
					log.Info().Msg("LazySave succeeded in EventList during context cancellation")
				}
				return
			case <-time.After(5 * time.Minute):
				// 检查是否需要同步
				el.Mutex.RLock()
				if el.IsSyncedWithFile {
					el.Mutex.RUnlock()
					log.Debug().Msg("LazySave skipped in EventList, Because we are lazy")
					continue
				}
				el.Mutex.RUnlock()

				// 同步
				el.Mutex.Lock()
				err := el.SaveNow()
				if err != nil {
					log.Warn().Err(err).Msg("LazySave failed in EventList. Some not important data may be lost")
				} else {
					log.Info().Msg("LazySave succeeded in EventList")
					el.IsSyncedWithFile = true
				}
				el.Mutex.Unlock()
			}
		}
	}()
}
