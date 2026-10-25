package main

import (
	"context"
	"sync"
)

// ライドID → ライドの最新状態（ride_statuses の created_at 最大）のプロセス内キャッシュ。
// ride_statuses への INSERT 後、トランザクションのコミットに成功した時点で更新する。
// 単一プロセスで動作する前提のため、複数台構成にする場合は見直しが必要。
var (
	rideLatestStatusesMu sync.RWMutex
	rideLatestStatuses   = map[string]string{}
)

// 初期データからライドごとの最新状態を読み込み、キャッシュを作り直す
func loadRideLatestStatuses(ctx context.Context) error {
	statuses := []RideStatus{}
	if err := db.SelectContext(ctx, &statuses, `SELECT ride_id, status FROM ride_statuses ORDER BY created_at ASC`); err != nil {
		return err
	}

	latest := make(map[string]string, len(statuses))
	for _, s := range statuses {
		// created_at の昇順に上書きし、最後に残ったものを最新とする
		latest[s.RideID] = s.Status
	}

	rideLatestStatusesMu.Lock()
	rideLatestStatuses = latest
	rideLatestStatusesMu.Unlock()
	return nil
}

func getRideLatestStatus(rideID string) (string, bool) {
	rideLatestStatusesMu.RLock()
	defer rideLatestStatusesMu.RUnlock()
	status, ok := rideLatestStatuses[rideID]
	return status, ok
}

func setRideLatestStatus(rideID string, status string) {
	rideLatestStatusesMu.Lock()
	defer rideLatestStatusesMu.Unlock()
	rideLatestStatuses[rideID] = status
}

// キャッシュミス時にDBから読んだ値を登録する。
// 読み込み後に別リクエストのコミットで新しい状態が登録されている場合は、それを古い値で上書きしない。
func setRideLatestStatusIfAbsent(rideID string, status string) {
	rideLatestStatusesMu.Lock()
	defer rideLatestStatusesMu.Unlock()
	if _, ok := rideLatestStatuses[rideID]; !ok {
		rideLatestStatuses[rideID] = status
	}
}
