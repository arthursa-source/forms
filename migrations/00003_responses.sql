-- Migration 3: respostas enviadas pelas páginas públicas.
--
-- Decisão importante: cada resposta guarda uma CÓPIA do label e do tipo do
-- campo no momento do envio, sem FK para form_fields. Editar um formulário
-- substitui a lista de campos, e um ON DELETE CASCADE apagaria respostas
-- antigas. Com a cópia, o histórico continua legível mesmo depois da edição.

-- +goose Up
CREATE TABLE responses (
    id         BIGSERIAL PRIMARY KEY,
    form_id    BIGINT      NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX responses_form_id_idx ON responses (form_id, created_at DESC);

CREATE TABLE response_answers (
    id          BIGSERIAL PRIMARY KEY,
    response_id BIGINT  NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    label       TEXT    NOT NULL,
    type        TEXT    NOT NULL,
    -- Valor normalizado como texto: "42" para number, "true"/"false" para
    -- checkbox. O tipo acima diz como interpretar.
    value       TEXT    NOT NULL,
    UNIQUE (response_id, position)
);

-- +goose Down
DROP TABLE response_answers;
DROP TABLE responses;
