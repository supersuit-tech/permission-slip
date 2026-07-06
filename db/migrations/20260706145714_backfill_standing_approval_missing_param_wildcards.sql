-- +goose Up

-- Standing approvals with only $meta (or other namespace) constraints omitted
-- action parameters, causing auto-approve to reject execution params as extra
-- keys. Backfill "*" for each schema property missing from stored constraints.

UPDATE standing_approvals sa
SET constraints = sa.constraints || wildcards.new_keys
FROM (
    SELECT
        sa2.standing_approval_id,
        jsonb_object_agg(prop_key, '"*"'::jsonb) AS new_keys
    FROM standing_approvals sa2
    INNER JOIN connector_actions ca ON ca.action_type = sa2.action_type
    CROSS JOIN LATERAL jsonb_object_keys(
        COALESCE(ca.parameters_schema->'properties', '{}'::jsonb)
    ) AS prop_key
    WHERE NOT (sa2.constraints ? prop_key)
    GROUP BY sa2.standing_approval_id
) wildcards
WHERE sa.standing_approval_id = wildcards.standing_approval_id;

-- +goose Down

-- Data-only migration; previous constraint shapes are not recoverable.
SELECT 1;
