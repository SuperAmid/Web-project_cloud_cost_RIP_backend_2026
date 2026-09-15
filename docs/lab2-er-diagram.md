# ER-диаграмма ЛР2

Визуальная версия для защиты: [lab2-er-diagram.svg](lab2-er-diagram.svg).

```text
provider_users
  id PK, email VARCHAR(120) UNIQUE, display_name VARCHAR(80), created_at TIMESTAMP
       1                                      1
       | creates                              | likes
       v                                      v
routers                                    router_likes
  id PK                                      router_id PK, FK -> routers.id
  name VARCHAR(120)                          user_id   PK, FK -> provider_users.id
  description VARCHAR(500)                   created_at TIMESTAMP
  status VARCHAR(12)
  image_url VARCHAR(500)
  video_url VARCHAR(500)
  router_type VARCHAR(24)
  throughput_mbps INTEGER
  power_consumption_w INTEGER
  port_count INTEGER
  location VARCHAR(160)
  master_router_name VARCHAR(120)
  created_at TIMESTAMP
  published_at TIMESTAMP NULL
  created_by_id FK -> provider_users.id
```

Все внешние ключи имеют `RESTRICT`: каскадное удаление запрещено. В StarUML перенесите эту схему в один `.mdj` файл: три класса/таблицы, первичные и внешние ключи, типы и связи 1:N (`provider_users → routers`) и M:N через `router_likes`.
