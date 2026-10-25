package main

import (
	"context"
	"sync"
	"time"
)

// 椅子の最新位置と累計移動距離
type chairLocation struct {
	Latitude               int
	Longitude              int
	TotalDistance          int
	TotalDistanceUpdatedAt time.Time
}

// 椅子ID → 最新位置・累計移動距離のプロセス内キャッシュ。
// chair_distances は初期データの読み込みにのみ使い、実行時の位置情報はこのキャッシュだけに記録する。
// 単一プロセスで動作する前提のため、複数台構成にする場合は見直しが必要。
var (
	chairLocationsMu sync.RWMutex
	chairLocations   = map[string]chairLocation{}
)

// 初期データから椅子ごとの最新位置・累計移動距離を読み込み、キャッシュを作り直す
func loadChairLocations(ctx context.Context) error {
	rows := []struct {
		ChairID                string    `db:"chair_id"`
		Latitude               int       `db:"latitude"`
		Longitude              int       `db:"longitude"`
		TotalDistance          int       `db:"total_distance"`
		TotalDistanceUpdatedAt time.Time `db:"total_distance_updated_at"`
	}{}
	if err := db.SelectContext(ctx, &rows, `SELECT chair_id, latitude, longitude, total_distance, total_distance_updated_at FROM chair_distances`); err != nil {
		return err
	}

	locations := make(map[string]chairLocation, len(rows))
	for _, r := range rows {
		locations[r.ChairID] = chairLocation{
			Latitude:               r.Latitude,
			Longitude:              r.Longitude,
			TotalDistance:          r.TotalDistance,
			TotalDistanceUpdatedAt: r.TotalDistanceUpdatedAt,
		}
	}

	chairLocationsMu.Lock()
	chairLocations = locations
	chairLocationsMu.Unlock()
	return nil
}

func getChairLocation(chairID string) (chairLocation, bool) {
	chairLocationsMu.RLock()
	defer chairLocationsMu.RUnlock()
	loc, ok := chairLocations[chairID]
	return loc, ok
}

// 最新位置を更新し、前回位置とのマンハッタン距離を累計移動距離に加算する（初回は 0）。
// 並行リクエストで加算が欠けないよう、読み出しから書き込みまでをロック内で行う
func updateChairLocation(chairID string, latitude, longitude int, recordedAt time.Time) {
	chairLocationsMu.Lock()
	defer chairLocationsMu.Unlock()
	loc, ok := chairLocations[chairID]
	if ok {
		loc.TotalDistance += calculateDistance(loc.Latitude, loc.Longitude, latitude, longitude)
	}
	loc.Latitude = latitude
	loc.Longitude = longitude
	loc.TotalDistanceUpdatedAt = recordedAt
	chairLocations[chairID] = loc
}
