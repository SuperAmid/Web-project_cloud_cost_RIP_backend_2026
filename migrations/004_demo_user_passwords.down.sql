UPDATE provider_router_users
SET password_hash = ''
WHERE email IN ('student@provider-routers.local', 'operator@provider-routers.local', 'viewer@provider-routers.local')
  AND password_hash = '$2a$10$qu9d1jJJ.Fg/2hczwKxGaexugeI7qLrKAZlJRyo.vtbrCwvEUHgva';
