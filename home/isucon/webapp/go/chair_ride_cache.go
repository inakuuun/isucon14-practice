package main

import (
	"context"
	"sync"
)

// 椅子ID → 椅子に割り当てられた最新ライド（updated_at 最大）のプロセス内キャッシュ。
// 椅子の最新ライドが変わるのはマッチング時の割り当てのみで、参照する項目（id, user_id, 座標）は作成後に変わらない。
// 単一プロセスで動作する前提のため、複数台構成にする場合は見直しが必要。
var (
	chairLatestRidesMu sync.RWMutex
	chairLatestRides   = map[string]*Ride{}
)

// 初期データから椅子ごとの最新ライドを読み込み、キャッシュを作り直す
func loadChairLatestRides(ctx context.Context) error {
	rides := []*Ride{}
	if err := db.SelectContext(ctx, &rides, `SELECT * FROM rides WHERE chair_id IS NOT NULL ORDER BY updated_at ASC`); err != nil {
		return err
	}

	latest := make(map[string]*Ride, len(rides))
	for _, ride := range rides {
		// updated_at の昇順に上書きし、最後に残ったものを最新とする
		latest[ride.ChairID.String] = ride
	}

	chairLatestRidesMu.Lock()
	chairLatestRides = latest
	chairLatestRidesMu.Unlock()
	return nil
}

func getChairLatestRide(chairID string) (*Ride, bool) {
	chairLatestRidesMu.RLock()
	defer chairLatestRidesMu.RUnlock()
	ride, ok := chairLatestRides[chairID]
	return ride, ok
}

func setChairLatestRide(chairID string, ride *Ride) {
	chairLatestRidesMu.Lock()
	defer chairLatestRidesMu.Unlock()
	chairLatestRides[chairID] = ride
}
