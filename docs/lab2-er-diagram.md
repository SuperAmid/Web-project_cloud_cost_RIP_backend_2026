# ER-диаграмма ЛР‑2: provider_routers

Визуальная версия: [lab2-er-diagram.svg](lab2-er-diagram.svg).

```text
provider_router_users
  id PK, email VARCHAR(120) UNIQUE, display_name VARCHAR(80), created_at TIMESTAMP
       1                                              1
       | creates                                      | likes
       v                                              v
provider_routers                               provider_router_likes
  id PK                                         provider_router_id PK, FK -> provider_routers.id
  name VARCHAR(120)                             user_id PK, FK -> provider_router_users.id
  description VARCHAR(500)                      created_at TIMESTAMP
  status VARCHAR(12)
  image_key VARCHAR(160)
  video_key VARCHAR(160)
  router_type VARCHAR(24)
  bandwidth_mbps INTEGER
  cost_rub INTEGER
  created_at TIMESTAMP
  published_at TIMESTAMP NULL
  created_by_id FK -> provider_router_users.id
```

Внешние ключи используют `RESTRICT`: каскадное удаление запрещено. В StarUML создайте один `.mdj` с тремя таблицами, полями, типами, PK/FK и связями 1:N (`provider_router_users → provider_routers`) и M:N через `provider_router_likes`.
