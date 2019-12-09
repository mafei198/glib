/*
The MIT License (MIT)

Copyright (c) 2018 SavinMax. All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package player

import (
	"context"
	"github.com/go-redis/redis"
	"github.com/mafei198/goslib/logger"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
	"time"
)

type GameData struct {
	Uuid       string
	Client     redis.UniversalClient
	Collection *mongo.Collection
}

const CacheExpire = 1 * time.Hour

func New(collection *mongo.Collection, client redis.UniversalClient) *GameData {
	return &GameData{
		Uuid:       xid.New().String(),
		Client:     client,
		Collection: collection,
	}
}

func (m *GameData) Start() error {
	return startPersister(m)
}

func (m *GameData) Stop() error {
	ensurePersistered(m.Uuid)
	return nil
}

func (m *GameData) Take(playerId string) (string, error) {
	logger.INFO("cache Take: ", playerId)
	content, err := m.getFromCache(playerId)
	if err == redis.Nil {
		content, err = m.getFromDB(playerId)
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		if err != nil {
			logger.ERR("Take PlayerData query DB failed: ", err)
			return "", err
		}
		return content, nil
	}

	if err != nil {
		logger.ERR("Take PlayerData from redis failed: ", playerId, err)
		return "", err
	}

	if err = m.delFromCache(playerId); err != nil {
		logger.ERR("cache_mgr del from redis failed: ", err)
	}

	return content, nil
}

func (m *GameData) Return(playerId, content string, version int64) (bool, error) {
	logger.INFO("cache Return: ", playerId)
	if err := m.persistToCache(playerId, content); err != nil {
		logger.ERR("Return PlayerData failed: ", playerId, err)
		return false, err
	}
	err := persistToDB(m.Uuid, playerId, content, version, true)
	return true, err
}

func (m *GameData) Persist(playerId, content string, version int64) (bool, error) {
	logger.INFO("cache Persist: ", playerId)
	err := persistToDB(m.Uuid, playerId, content, version, false)
	return true, err
}

func (m *GameData) cacheKey(playerId string) string {
	return strings.Join([]string{"player_data", playerId}, ":")
}

func (m *GameData) getFromDB(playerId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.D{
		{"_id", playerId},
	}
	single := m.Collection.FindOne(ctx, filter)
	err := single.Err()
	if err != nil {
		return "", single.Err()
	}
	var result bson.M
	if err := single.Decode(&result); err != nil {
		if err.Error() == mongo.ErrNoDocuments.Error() {
			return "", nil
		} else {
			return "", err
		}
	}
	content := result["Content"].(string)
	return content, nil
}

func (m *GameData) persistToCache(playerId, content string) error {
	key := m.cacheKey(playerId)
	_, err := m.Client.Set(key, content, 0).Result()
	return err
}

func (m *GameData) getFromCache(playerId string) (string, error) {
	key := m.cacheKey(playerId)
	return m.Client.Get(key).Result()
}

func (m *GameData) delFromCache(playerId string) error {
	key := m.cacheKey(playerId)
	_, err := m.Client.Del(key).Result()
	return err
}
