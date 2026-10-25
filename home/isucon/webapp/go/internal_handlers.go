package main

import (
	"database/sql"
	"net/http"
	"sync"
)

// 同じ椅子を二重に割り当てないよう、マッチングは同時に1つしか実行しない
var matchingMu sync.Mutex

type matchingCandidate struct {
	chairID   string
	latitude  int
	longitude int
	speed     int
}

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
// 待ちが長いライドから順に、乗車地点への到着時間（マンハッタン距離 ÷ 速度）が最短の空き椅子を割り当てる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	matchingMu.Lock()
	defer matchingMu.Unlock()

	rides := []*Ride{}
	if err := db.SelectContext(ctx, &rides, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at`); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(rides) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 稼働中で位置があり、最新ライドがないか最新ライドの COMPLETED が椅子に通知済みの椅子を候補とする
	candidates := []matchingCandidate{}
	for _, chair := range listCachedChairs() {
		if !chair.IsActive {
			continue
		}
		loc, ok := getChairLocation(chair.ID)
		if !ok {
			continue
		}
		if ride, ok := getChairLatestRide(chair.ID); ok && !isRideCompletedSentToChair(ride.ID) {
			continue
		}
		candidates = append(candidates, matchingCandidate{
			chairID:   chair.ID,
			latitude:  loc.Latitude,
			longitude: loc.Longitude,
			speed:     getChairModelSpeed(chair.Model),
		})
	}

	for _, ride := range rides {
		if len(candidates) == 0 {
			break
		}

		// 距離/速度 の大小を、浮動小数を使わず 距離a*速度b と 距離b*速度a の比較で求める
		best := -1
		bestDistance, bestSpeed := 0, 1
		for i, c := range candidates {
			distance := calculateDistance(c.latitude, c.longitude, ride.PickupLatitude, ride.PickupLongitude)
			if best == -1 || distance*bestSpeed < bestDistance*c.speed {
				best, bestDistance, bestSpeed = i, distance, c.speed
			}
		}
		matched := candidates[best]

		if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.chairID, ride.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		ride.ChairID = sql.NullString{String: matched.chairID, Valid: true}
		setChairLatestRide(matched.chairID, ride)

		candidates[best] = candidates[len(candidates)-1]
		candidates = candidates[:len(candidates)-1]
	}

	w.WriteHeader(http.StatusNoContent)
}
