package misc

import (
	types "github.com/AsenHu/mewlink/internal/upgrader/v1"
	"github.com/rs/zerolog/log"
)

func IsGoodRoomInfo(info *types.RoomInfo) bool {
	if info.GetRoomID() == "" {
		log.Error().Msg("RoomID not found, this should not happen, database corrupted")
		return false
	}
	if info.GetChatID() == 0 {
		log.Error().Msg("ChatID not found, this should not happen, database corrupted")
		return false
	}
	return true
}
