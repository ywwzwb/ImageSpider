-- Database Migration Script v1
-- Add integrity_status column to images table
-- Handles partitioned tables properly

-- 1. Create migrations tracking table if not exists
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Check if migration already applied
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM schema_migrations WHERE version = 'v1'
    ) THEN
        RAISE NOTICE 'Migration v1 already applied, skipping...';
        RETURN;
    END IF;

    -- 3. Add integrity_status column to the main table (if not exists)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'images'
        AND column_name = 'integrity_status'
    ) THEN
        ALTER TABLE images ADD COLUMN integrity_status SMALLINT DEFAULT 0;
        RAISE NOTICE 'Added integrity_status column to images table';
    ELSE
        RAISE NOTICE 'integrity_status column already exists in images table';
    END IF;

    -- 4. Add integrity_status column to all existing partition tables
    DECLARE
        partition_record RECORD;
    BEGIN
        -- Loop through all partitions of the images table
        FOR partition_record IN
            SELECT
                nmsp_parent.nspname AS parent_schema,
                parent.relname AS parent_table,
                nmsp_child.nspname AS child_schema,
                child.relname AS child_table
            FROM pg_inherits
            JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
            JOIN pg_class child ON pg_inherits.inhrelid = child.oid
            JOIN pg_namespace nmsp_parent ON parent.relnamespace = nmsp_parent.oid
            JOIN pg_namespace nmsp_child ON child.relnamespace = nmsp_child.oid
            WHERE parent.relname = 'images'
            AND nmsp_parent.nspname = 'public'
        LOOP
            -- Check if integrity_status column exists in this partition
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_schema = partition_record.child_schema
                AND table_name = partition_record.child_table
                AND column_name = 'integrity_status'
            ) THEN
                -- Add column to partition
                EXECUTE format(
                    'ALTER TABLE %I.%I ADD COLUMN integrity_status SMALLINT DEFAULT 0',
                    partition_record.child_schema,
                    partition_record.child_table
                );
                RAISE NOTICE 'Added integrity_status column to partition: %', partition_record.child_table;
            END IF;
        END LOOP;
    END;

    -- 5. Set default value for existing columns
    EXECUTE 'UPDATE images SET integrity_status = 0 WHERE integrity_status IS NULL';
    RAISE NOTICE 'Set default values for existing records';

    -- 6. Record migration as applied
    INSERT INTO schema_migrations (version) VALUES ('v1');
    RAISE NOTICE 'Migration v1 completed successfully';

END $$;
