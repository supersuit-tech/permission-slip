-- +goose Up

-- Backfill operation_type for rows that still have the column default from
-- 20260413145754_add_operation_type_to_connector_actions. Uses the same
-- verb-segment rules as connectors.InferOperationType (delete > read > write).
CREATE OR REPLACE FUNCTION _tmp_infer_operation_type(p_action_type text)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
  dot_pos int;
  suffix text;
  seg text;
  segs text[];
BEGIN
  dot_pos := strpos(p_action_type, '.');
  IF dot_pos = 0 OR dot_pos = length(p_action_type) THEN
    RETURN 'write';
  END IF;
  suffix := substring(p_action_type from dot_pos + 1);
  IF suffix = '' THEN
    RETURN 'write';
  END IF;

  segs := string_to_array(suffix, '_');
  FOREACH seg IN ARRAY segs
  LOOP
    IF seg IS NULL OR seg = '' THEN
      CONTINUE;
    END IF;

    IF seg IN (
      'delete', 'remove', 'cancel', 'close', 'archive', 'purge', 'void',
      'unfollow', 'unlike', 'unretweet'
    ) THEN
      RETURN 'delete';
    END IF;

    IF seg IN (
      'get', 'list', 'read', 'search', 'describe', 'query', 'check',
      'download', 'export', 'fetch', 'view', 'price'
    ) THEN
      RETURN 'read';
    END IF;

    IF seg IN (
      'create', 'update', 'send', 'post', 'put', 'patch', 'add', 'merge',
      'trigger', 'upload', 'share', 'move', 'complete', 'transition', 'assign',
      'invite', 'set', 'schedule', 'fulfill', 'enroll', 'reply', 'retweet',
      'like', 'follow', 'swap', 'book', 'issue', 'adjust', 'record', 'convert',
      'run', 'append', 'write', 'tag', 'reconcile'
    ) THEN
      RETURN 'write';
    END IF;
  END LOOP;

  RETURN 'write';
END;
$$;

-- Rows with operation_type = 'write' are indistinguishable from the column default added in
-- 20260413145754. At the time of this backfill, no manifest had yet persisted a different
-- explicit override as 'write', so inferring from action_type is safe. If this migration is
-- delayed after connectors start storing explicit overrides, revisit before applying.
UPDATE connector_actions
SET operation_type = _tmp_infer_operation_type(action_type)
WHERE operation_type = 'write';

DROP FUNCTION _tmp_infer_operation_type(text);

-- +goose Down

-- Data-only migration; previous operation_type values are not recoverable.
SELECT 1;
