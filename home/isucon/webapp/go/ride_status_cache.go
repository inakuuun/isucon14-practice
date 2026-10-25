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

// COMPLETED の通知が椅子に届いたライドIDの集合。
// 椅子への通知は古い状態から順に送るため、COMPLETED が届いていればそのライドの全状態が通知済みとなる。
var (
	rideCompletedSentToChairMu sync.RWMutex
	rideCompletedSentToChair   = map[string]struct{}{}
)

// 初期データから COMPLETED を椅子に通知済みのライドを読み込み、キャッシュを作り直す
func loadRideCompletedSentToChair(ctx context.Context) error {
	rideIDs := []string{}
	if err := db.SelectContext(ctx, &rideIDs, `SELECT ride_id FROM ride_statuses WHERE status = 'COMPLETED' AND chair_sent_at IS NOT NULL`); err != nil {
		return err
	}

	sent := make(map[string]struct{}, len(rideIDs))
	for _, rideID := range rideIDs {
		sent[rideID] = struct{}{}
	}

	rideCompletedSentToChairMu.Lock()
	rideCompletedSentToChair = sent
	rideCompletedSentToChairMu.Unlock()
	return nil
}

func isRideCompletedSentToChair(rideID string) bool {
	rideCompletedSentToChairMu.RLock()
	defer rideCompletedSentToChairMu.RUnlock()
	_, ok := rideCompletedSentToChair[rideID]
	return ok
}

func setRideCompletedSentToChair(rideID string) {
	rideCompletedSentToChairMu.Lock()
	defer rideCompletedSentToChairMu.Unlock()
	rideCompletedSentToChair[rideID] = struct{}{}
}
