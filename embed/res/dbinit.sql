--创建主表
CREATE TABLE IF NOT EXISTS images (
    id TEXT NOT NULL,
    source_id TEXT NOT NULL,
    tags TEXT[],
    image_url TEXT,
    local_path TEXT,
    post_time TIMESTAMP NOT null,
    PRIMARY KEY (id, source_id, post_time)
) PARTITION BY LIST (source_id);

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
