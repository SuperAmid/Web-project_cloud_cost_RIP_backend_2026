-- ЛР3: регистрация пользователей — храним bcrypt-хеш пароля (60 символов).
ALTER TABLE provider_router_users
  ADD COLUMN IF NOT EXISTS password_hash VARCHAR(72) NOT NULL DEFAULT '';
