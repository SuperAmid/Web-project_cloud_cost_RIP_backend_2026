-- ЛР4: у пользователей из начальных данных появляется пароль для входа (JWT).
-- Пароль у всех трёх: password123 (bcrypt). Уже заданные пароли не меняются.
UPDATE provider_router_users
SET password_hash = '$2a$10$qu9d1jJJ.Fg/2hczwKxGaexugeI7qLrKAZlJRyo.vtbrCwvEUHgva'
WHERE email IN ('student@provider-routers.local', 'operator@provider-routers.local', 'viewer@provider-routers.local')
  AND password_hash = '';
