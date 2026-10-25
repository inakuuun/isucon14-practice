package main

import (
	"context"
	"sync"
)

// 椅子モデル名 → 移動速度のプロセス内キャッシュ。chair_models はマスターデータで、実行時に変わらない。
var (
	chairModelSpeedsMu sync.RWMutex
	chairModelSpeeds   = map[string]int{}
)

// 初期データから椅子モデルの速度を読み込み、キャッシュを作り直す
func loadChairModelSpeeds(ctx context.Context) error {
	models := []ChairModel{}
	if err := db.SelectContext(ctx, &models, `SELECT name, speed FROM chair_models`); err != nil {
		return err
	}

	speeds := make(map[string]int, len(models))
	for _, m := range models {
		speeds[m.Name] = m.Speed
	}

	chairModelSpeedsMu.Lock()
	chairModelSpeeds = speeds
	chairModelSpeedsMu.Unlock()
	return nil
}

// モデルの速度を返す。未知のモデルは最低速度 1 として扱う
func getChairModelSpeed(model string) int {
	chairModelSpeedsMu.RLock()
	defer chairModelSpeedsMu.RUnlock()
	if speed, ok := chairModelSpeeds[model]; ok && speed > 0 {
		return speed
	}
	return 1
}
