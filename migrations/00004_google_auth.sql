-- Migration 4: login com Google.
-- Uma conta criada pelo Google não tem senha, então password_hash passa a
-- aceitar NULL. google_id guarda o "sub" devolvido pelo Google: é o id
-- estável da conta (o e-mail pode mudar).

-- +goose Up
ALTER TABLE users ADD COLUMN google_id TEXT UNIQUE;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- Uma conta precisa de pelo menos uma forma de entrar: senha ou Google.
ALTER TABLE users ADD CONSTRAINT users_login_method_check
    CHECK (password_hash IS NOT NULL OR google_id IS NOT NULL);

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_login_method_check;
DELETE FROM users WHERE password_hash IS NULL;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
ALTER TABLE users DROP COLUMN google_id;
