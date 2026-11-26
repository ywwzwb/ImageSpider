--创建主表
CREATE TABLE IF NOT EXISTS images (
    id TEXT NOT NULL,
    source_id TEXT NOT NULL,
    tags TEXT[],
    image_url TEXT,
    local_path TEXT,
    post_time TIMESTAMP NOT null,
    integrity_status SMALLINT DEFAULT 0, -- 0=未知, 1=好图, -1=坏图; NULL也视为未知
    download_attempts INT NOT NULL DEFAULT 0, -- 连续下载失败次数
    next_download_at TIMESTAMPTZ, -- 下次允许重试的时间；NULL 表示立即可试
    PRIMARY KEY (id, source_id, post_time)
) PARTITION BY LIST (source_id);

-- 兼容已存在的数据库：补充下载重试字段（分区表会自动级联到所有分区）
ALTER TABLE images ADD COLUMN IF NOT EXISTS download_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE images ADD COLUMN IF NOT EXISTS next_download_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS tags (
    source_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    count INT,
    cover TEXT,
    PRIMARY KEY (source_id, tag)
) PARTITION BY LIST (source_id);

--创建索引
CREATE INDEX IF NOT EXISTS idx_images_id ON images (id);
CREATE INDEX IF NOT EXISTS idx_images_tags ON images USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_images_source_id ON images (source_id);
CREATE INDEX IF NOT EXISTS idx_images_post_time ON images (post_time);
