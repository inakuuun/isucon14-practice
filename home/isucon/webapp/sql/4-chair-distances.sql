-- 初期データの chair_locations から椅子ごとの累計移動距離と最新位置を生成する
INSERT INTO chair_distances (chair_id, latitude, longitude, total_distance, total_distance_updated_at)
SELECT chair_id, latitude, longitude, total_distance, created_at
FROM (SELECT chair_id,
             latitude,
             longitude,
             created_at,
             SUM(IFNULL(distance, 0)) OVER (PARTITION BY chair_id)                     AS total_distance,
             ROW_NUMBER() OVER (PARTITION BY chair_id ORDER BY created_at DESC, id DESC) AS rn
      FROM (SELECT id,
                   chair_id,
                   latitude,
                   longitude,
                   created_at,
                   ABS(latitude - LAG(latitude) OVER (PARTITION BY chair_id ORDER BY created_at)) +
                   ABS(longitude - LAG(longitude) OVER (PARTITION BY chair_id ORDER BY created_at)) AS distance
            FROM chair_locations) tmp) latest
WHERE rn = 1;
