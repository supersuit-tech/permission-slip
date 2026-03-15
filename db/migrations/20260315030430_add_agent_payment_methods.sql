-- +goose Up

-- Join table linking agents to payment methods.
-- Each agent can have at most one assigned payment method (PRIMARY KEY on agent_id),
-- but one payment method can be shared across multiple agents.
CREATE TABLE agent_payment_methods (
    agent_id          BIGINT      NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    payment_method_id UUID        NOT NULL REFERENCES payment_methods(id) ON DELETE CASCADE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id)
);

CREATE INDEX idx_agent_payment_methods_pm ON agent_payment_methods(payment_method_id);

ALTER TABLE agent_payment_methods ENABLE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_backend') THEN
        CREATE POLICY app_backend_all ON agent_payment_methods
            FOR ALL TO app_backend USING (true) WITH CHECK (true);
        GRANT SELECT, INSERT, UPDATE, DELETE ON agent_payment_methods TO app_backend;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down

DROP TABLE IF EXISTS agent_payment_methods;
