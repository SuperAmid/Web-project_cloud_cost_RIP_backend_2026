package repository

import (
	"context"
	"errors"
	"mime/multipart"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"provider_routers/internal/ds"
)

// ProviderRouterRow — опубликованный маршрутизатор вместе с вычисленными в БД
// полями: числом лайков, лайком текущего пользователя и именем создателя.
type ProviderRouterRow struct {
	ID            uint
	Name          string
	Description   string
	Status        string
	ImageKey      string
	VideoKey      string
	RouterType    string
	BandwidthMbps int
	CostRub       int
	CreatedAt     time.Time
	PublishedAt   *time.Time
	CreatedByID   uint
	CreatorName   string
	LikesCount    int64
	IsLiked       bool
}

// ProviderRouterFilter — фильтры плитки; nil означает «не фильтровать».
type ProviderRouterFilter struct {
	MinBandwidthMbps *int
	MaxCostRub       *int
}

// ProviderRouterPublish — поля, которые создатель заполняет при публикации.
type ProviderRouterPublish struct {
	Description   string
	RouterType    string
	BandwidthMbps int
	CostRub       int
}

// publishedProviderRouters — общий запрос: только опубликованные, с подсчётом лайков в БД.
func (r *Repository) publishedProviderRouters(userID uint) *gorm.DB {
	return r.db.Table("provider_routers AS pr").
		Select(`pr.id, pr.name, pr.description, pr.status, pr.image_key, pr.video_key,
			pr.router_type, pr.bandwidth_mbps, pr.cost_rub, pr.created_at, pr.published_at,
			pr.created_by_id, u.display_name AS creator_name,
			(SELECT COUNT(*) FROM provider_router_likes l WHERE l.provider_router_id = pr.id) AS likes_count,
			EXISTS (SELECT 1 FROM provider_router_likes l WHERE l.provider_router_id = pr.id AND l.user_id = ?) AS is_liked`, userID).
		Joins("JOIN provider_router_users u ON u.id = pr.created_by_id").
		Where("pr.status = ?", ds.ProviderRouterStatusPublished)
}

// ListProviderRouters — плитка с фильтрацией на стороне БД.
func (r *Repository) ListProviderRouters(userID uint, filter ProviderRouterFilter) ([]ProviderRouterRow, error) {
	query := r.publishedProviderRouters(userID)
	if filter.MinBandwidthMbps != nil {
		query = query.Where("pr.bandwidth_mbps >= ?", *filter.MinBandwidthMbps)
	}
	if filter.MaxCostRub != nil {
		query = query.Where("pr.cost_rub <= ?", *filter.MaxCostRub)
	}
	rows := []ProviderRouterRow{}
	return rows, query.Order("pr.id").Scan(&rows).Error
}

// GetFeedProviderRouter возвращает ровно одну строку ленты из БД:
// без id — первую; с id — эту; с id и next — следующую после id (по кругу).
func (r *Repository) GetFeedProviderRouter(userID uint, id *uint, next bool) (ProviderRouterRow, error) {
	query := r.publishedProviderRouters(userID).Order("pr.id").Limit(1)
	switch {
	case id != nil && next:
		query = query.Where("pr.id > ?", *id)
	case id != nil:
		query = query.Where("pr.id = ?", *id)
	}
	var rows []ProviderRouterRow
	if err := query.Scan(&rows).Error; err != nil {
		return ProviderRouterRow{}, err
	}
	if len(rows) == 0 && id != nil && next {
		// после последней карточки лента начинается сначала
		if err := r.publishedProviderRouters(userID).Order("pr.id").Limit(1).Scan(&rows).Error; err != nil {
			return ProviderRouterRow{}, err
		}
	}
	if len(rows) == 0 {
		return ProviderRouterRow{}, ErrNotFound
	}
	return rows[0], nil
}

// GetDraftProviderRouter — черновик пользователя (не более одного, id не нужен).
func (r *Repository) GetDraftProviderRouter(userID uint) (ds.ProviderRouter, error) {
	var providerRouter ds.ProviderRouter
	err := r.db.Where("created_by_id = ? AND status = ?", userID, ds.ProviderRouterStatusDraft).First(&providerRouter).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return providerRouter, ErrNotFound
	}
	return providerRouter, err
}

// CreateDraftProviderRouter создаёт черновик и сохраняет фото и видео в MinIO.
func (r *Repository) CreateDraftProviderRouter(ctx context.Context, userID uint, name string, image, video *multipart.FileHeader) (ds.ProviderRouter, error) {
	if _, err := r.GetDraftProviderRouter(userID); err == nil {
		return ds.ProviderRouter{}, ErrConflict
	} else if !errors.Is(err, ErrNotFound) {
		return ds.ProviderRouter{}, err
	}
	imageKey, err := r.media.Upload(ctx, ProviderRouterImage, image)
	if err != nil {
		return ds.ProviderRouter{}, err
	}
	videoKey, err := r.media.Upload(ctx, ProviderRouterVideo, video)
	if err != nil {
		r.media.Remove(ctx, imageKey)
		return ds.ProviderRouter{}, err
	}
	providerRouter := ds.ProviderRouter{
		Name:        name,
		Status:      ds.ProviderRouterStatusDraft,
		ImageKey:    imageKey,
		VideoKey:    videoKey,
		RouterType:  "residential",
		CreatedByID: userID,
	}
	if err := r.db.Create(&providerRouter).Error; err != nil {
		r.media.Remove(ctx, imageKey)
		r.media.Remove(ctx, videoKey)
		if errors.Is(err, gorm.ErrDuplicatedKey) { // сработал индекс «один черновик на создателя»
			return ds.ProviderRouter{}, ErrConflict
		}
		return ds.ProviderRouter{}, err
	}
	return providerRouter, nil
}

// lockOwnProviderRouter блокирует строку на время транзакции и проверяет владельца.
func lockOwnProviderRouter(tx *gorm.DB, userID, id uint) (ds.ProviderRouter, error) {
	var providerRouter ds.ProviderRouter
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND status <> ?", id, ds.ProviderRouterStatusDeleted).
		First(&providerRouter).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return providerRouter, ErrNotFound
	}
	if err != nil {
		return providerRouter, err
	}
	if providerRouter.CreatedByID != userID {
		return providerRouter, ErrForbidden
	}
	return providerRouter, nil
}

// PublishProviderRouter: draft -> published. Дата публикации ставится на сервере.
func (r *Repository) PublishProviderRouter(userID, id uint, fields ProviderRouterPublish) (ds.ProviderRouter, error) {
	var providerRouter ds.ProviderRouter
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		if providerRouter, err = lockOwnProviderRouter(tx, userID, id); err != nil {
			return err
		}
		if providerRouter.Status != ds.ProviderRouterStatusDraft {
			return ErrConflict
		}
		now := time.Now()
		return tx.Model(&providerRouter).Updates(map[string]any{
			"description":    fields.Description,
			"router_type":    fields.RouterType,
			"bandwidth_mbps": fields.BandwidthMbps,
			"cost_rub":       fields.CostRub,
			"status":         ds.ProviderRouterStatusPublished,
			"published_at":   now,
		}).Error
	})
	return providerRouter, err
}

// DeleteProviderRouter — логическое удаление своей записи: draft|published -> deleted.
func (r *Repository) DeleteProviderRouter(userID, id uint) (ds.ProviderRouter, error) {
	var providerRouter ds.ProviderRouter
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		if providerRouter, err = lockOwnProviderRouter(tx, userID, id); err != nil {
			return err
		}
		return tx.Model(&providerRouter).Update("status", ds.ProviderRouterStatusDeleted).Error
	})
	return providerRouter, err
}

// SetProviderRouterLike ставит (like=true) или снимает лайк текущего пользователя.
func (r *Repository) SetProviderRouterLike(userID, id uint, like bool) (int64, error) {
	var likesCount int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&ds.ProviderRouter{}).Where("id = ? AND status = ?", id, ds.ProviderRouterStatusPublished).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return ErrNotFound
		}
		likeRow := ds.ProviderRouterLike{ProviderRouterID: id, UserID: userID}
		if like {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&likeRow).Error; err != nil {
				return err
			}
		} else if err := tx.Where("provider_router_id = ? AND user_id = ?", id, userID).Delete(&ds.ProviderRouterLike{}).Error; err != nil {
			return err
		}
		return tx.Model(&ds.ProviderRouterLike{}).Where("provider_router_id = ?", id).Count(&likesCount).Error
	})
	return likesCount, err
}

// MediaURL — публичная ссылка на файл маршрутизатора в MinIO.
func (r *Repository) MediaURL(key, fallback string) string { return r.media.URL(key, fallback) }
