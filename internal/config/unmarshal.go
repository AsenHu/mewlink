package config

import (
	"encoding/json"
	"fmt"

	v0 "github.com/AsenHu/mewlink/internal/upgrader/v0"
	v1 "github.com/AsenHu/mewlink/internal/upgrader/v1"
	"github.com/rs/zerolog/log"
)

const CONFIG_VERSION = 1

func (c *Config) unmarshal(data []byte) (err error) {
	// 反序列化配置，配置可能是任意的 JSON 格式
	rawData := make(map[string]interface{})
	if err = json.Unmarshal(data, &rawData); err != nil {
		return
	}
	// 检查版本号是否存在
	var version uint8
	if rawData["version"] == nil {
		// 如果版本号不存在，设置为 0
		version = uint8(0)
	} else {
		// 如果版本号存在，将版本号转换为 uint8
		versionFloat, ok := rawData["version"].(float64)
		if !ok {
			err = fmt.Errorf("Config version is not a number")
			return
		}
		version = uint8(versionFloat)
	}

	log.Debug().
		Uint8("version", version).
		Msg("Config version")

	// 检查版本号是否是最新版本
	// 如果是最新版本，将配置赋值给 Config
	if version == CONFIG_VERSION {
		log.Debug().Msg("Config version is the latest")
		if err = json.Unmarshal(data, &c.Content); err != nil {
			return err
		}
		return
	}
	// 如果比最新版本高，返回错误
	if version > CONFIG_VERSION {
		log.Error().Msg("Config version is too high")
		return fmt.Errorf("Config version is too high")
	}
	// 如果比最新版本低，进行升级
	if version == 0 {
		// 版本 0 升级到版本 1
		var v0Data v0.Content
		var v1Data v1.Content
		err = json.Unmarshal(data, &v0Data)
		if err != nil {
			return
		}
		log.Debug().Msg("Upgrading config from version 0 to 1")
		v1Data, err = ver0to1(v0Data)
		if err != nil {
			return
		}

		// 这是最后一个版本，所以直接赋值
		c.Content = v1Data
	}
	log.Debug().
		Uint8("newVersion", c.Content.Version).
		Msg("Finished upgrading config")
	return
}
