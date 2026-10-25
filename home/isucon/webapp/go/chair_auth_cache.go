package main

import (
	"context"
	"sync"
)

// アクセストークン → 椅子のプロセス内キャッシュ。
// access_token は椅子登録時の INSERT 以降変わらず、chairs への UPDATE は is_active のみ。
// キャッシュ内の *Chair はリクエストのコンテキストに渡すため書き換えず、更新時は複製して差し替える。
// 単一プロセスで動作する前提のため、複数台構成にする場合は見直しが必要。
var (
	chairsByAccessTokenMu sync.RWMutex
	chairsByAccessToken   = map[string]*Chair{}
	chairAccessTokenByID  = map[string]string{}
)

// 初期データから椅子を読み込み、キャッシュを作り直す
func loadChairsByAccessToken(ctx context.Context) error {
	chairs := []*Chair{}
	if err := db.SelectContext(ctx, &chairs, `SELECT * FROM chairs`); err != nil {
		return err
	}

	byToken := make(map[string]*Chair, len(chairs))
	tokenByID := make(map[string]string, len(chairs))
	for _, chair := range chairs {
		byToken[chair.AccessToken] = chair
		tokenByID[chair.ID] = chair.AccessToken
	}

	chairsByAccessTokenMu.Lock()
	chairsByAccessToken = byToken
	chairAccessTokenByID = tokenByID
	chairsByAccessTokenMu.Unlock()
	return nil
}

func getChairByAccessToken(accessToken string) (*Chair, bool) {
	chairsByAccessTokenMu.RLock()
	defer chairsByAccessTokenMu.RUnlock()
	chair, ok := chairsByAccessToken[accessToken]
	return chair, ok
}

func setChairByAccessToken(chair *Chair) {
	chairsByAccessTokenMu.Lock()
	defer chairsByAccessTokenMu.Unlock()
	chairsByAccessToken[chair.AccessToken] = chair
	chairAccessTokenByID[chair.ID] = chair.AccessToken
}

// キャッシュ済みの椅子の is_active を更新する。未キャッシュの場合は何もしない（次回の認証時にDBから読み込まれる）
func setChairIsActive(chairID string, isActive bool) {
	chairsByAccessTokenMu.Lock()
	defer chairsByAccessTokenMu.Unlock()
	accessToken, ok := chairAccessTokenByID[chairID]
	if !ok {
		return
	}
	chair, ok := chairsByAccessToken[accessToken]
	if !ok {
		return
	}
	updated := *chair
	updated.IsActive = isActive
	chairsByAccessToken[accessToken] = &updated
}

// キャッシュ済みの全椅子を返す。返すスライスは呼び出し側で並べ替えてよい（要素の *Chair は書き換えない）
func listCachedChairs() []*Chair {
	chairsByAccessTokenMu.RLock()
	defer chairsByAccessTokenMu.RUnlock()
	chairs := make([]*Chair, 0, len(chairsByAccessToken))
	for _, chair := range chairsByAccessToken {
		chairs = append(chairs, chair)
	}
	return chairs
}
