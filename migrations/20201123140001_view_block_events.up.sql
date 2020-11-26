CREATE TABLE view_block_events (
   id BIGSERIAL,
   block_height BIGINT,
   block_hash VARCHAR NOT NULL,
   block_time BIGINT NOT NULL,
   data JSONB NOT NULL,
   PRIMARY KEY(id)
);

CREATE INDEX view_block_events_block_height_index ON view_block_events(block_height);

CREATE INDEX view_block_events_order_by_block_height_desc_index ON view_block_events(block_height DESC);
