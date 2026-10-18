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

// ProviderRouterFilter — фильтры плитки; nil означает «не фильтровать».
type ProviderRouterFilter struct {
	MinBandwidthMbps *int
	MaxCostRub       *int
}

// ProviderRouterPublish — поля, которые создатель заполняет при публикации.
type ProviderRouterPublish struct {
	Description   string
	BandwidthMbps int
	CostRub       int
}

// published — общий запрос через ORM: только опубликованные, с лайками и создателем.
func (r *Repository) published() *gorm.DB {
	return r.db.Model(&ds.ProviderRouter{}).
		Preload("Likes").
		Preload("CreatedBy").
		Where("status = ?", ds.ProviderRouterStatusPublished).
		Order("id")
}

// ListProviderRouters — плитка; фильтры по полям по теме выполняются в БД.
func (r *Repository) ListProviderRouters(filter ProviderRouterFilter) ([]ds.ProviderRouter, error) {
	query := r.published()
	if filter.MinBandwidthMbps != nil {
		query = query.Where("bandwidth_mbps >= ?", *filter.MinBandwidthMbps)
	}
	if filter.MaxCostRub != nil {
		query = query.Where("cost_rub <= ?", *filter.MaxCostRub)
	}
	items := []ds.ProviderRouter{}
	return items, query.Find(&items).Error
}

// GetFeedProviderRouter возвращает одну запись ленты (First → LIMIT 1):
// без id — первую; с id — эту; с id и next — следующую после id (после последней — первую).
func (r *Repository) GetFeedProviderRouter(id *uint, next bool) (ds.ProviderRouter, error) {
	var item ds.ProviderRouter
	var err error
	switch {
	case id == nil:
		err = r.published().First(&item).Error
	case next:
		err = r.published().Where("id > ?", *id).First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = r.published().First(&item).Error
		}
	default:
		err = r.published().Where("id = ?", *id).First(&item).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return item, ErrNotFound
	}
	return item, err
}

// GetDraftProviderRouter — черновик пользователя (не более одного, id не нужен).
func (r *Repository) GetDraftProviderRouter(userID uint) (ds.ProviderRouter, error) {
	var item ds.ProviderRouter
	err := r.db.Where("created_by_id = ? AND status = ?", userID, ds.ProviderRouterStatusDraft).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return item, ErrNotFound
	}
	return item, err
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
	item := ds.ProviderRouter{
		Name:        name,
		Status:      ds.ProviderRouterStatusDraft,
		ImageKey:    imageKey,
		VideoKey:    videoKey,
		CreatedByID: userID,
	}
	if err := r.db.Create(&item).Error; err != nil {
		r.media.Remove(ctx, imageKey)
		r.media.Remove(ctx, videoKey)
		if errors.Is(err, gorm.ErrDuplicatedKey) { // индекс «один черновик на создателя»
			return ds.ProviderRouter{}, ErrConflict
		}
		return ds.ProviderRouter{}, err
	}
	return item, nil
}

// lockOwnProviderRouter блокирует строку (SELECT … FOR UPDATE) и проверяет владельца.
func lockOwnProviderRouter(tx *gorm.DB, userID, id uint) (ds.ProviderRouter, error) {
	var item ds.ProviderRouter
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND status <> ?", id, ds.ProviderRouterStatusDeleted).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	if item.CreatedByID != userID {
		return item, ErrForbidden
	}
	return item, nil
}

// PublishProviderRouter: draft -> published. Дата публикации ставится на сервере.
func (r *Repository) PublishProviderRouter(userID, id uint, fields ProviderRouterPublish) (ds.ProviderRouter, error) {
	var item ds.ProviderRouter
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		if item, err = lockOwnProviderRouter(tx, userID, id); err != nil {
			return err
		}
		if item.Status != ds.ProviderRouterStatusDraft {
			return ErrConflict
		}
		now := time.Now()
		item.Description = &fields.Description
		item.BandwidthMbps = &fields.BandwidthMbps
		item.CostRub = &fields.CostRub
		item.Status = ds.ProviderRouterStatusPublished
		item.PublishedAt = &now
		return tx.Model(&item).Select("description", "bandwidth_mbps", "cost_rub", "status", "published_at").Updates(&item).Error
	})
	return item, err
}

// DeleteProviderRouter — логическое удаление своей записи: draft|published -> deleted.
func (r *Repository) DeleteProviderRouter(userID, id uint) (ds.ProviderRouter, error) {
	var item ds.ProviderRouter
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		if item, err = lockOwnProviderRouter(tx, userID, id); err != nil {
			return err
		}
		item.Status = ds.ProviderRouterStatusDeleted
		return tx.Model(&item).Update("status", item.Status).Error
	})
	return item, err
}

// SetProviderRouterLike ставит (like=true) или снимает лайк текущего пользователя.
func (r *Repository) SetProviderRouterLike(userID, id uint, like bool) (int64, error) {
	var likesCount int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var item ds.ProviderRouter
		err := tx.Where("id = ? AND status = ?", id, ds.ProviderRouterStatusPublished).First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if like {
			row := ds.ProviderRouterLike{ProviderRouterID: id, UserID: userID}
			err = tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "provider_router_id"}, {Name: "user_id"}},
				DoNothing: true,
			}).Create(&row).Error
		} else {
			err = tx.Where("provider_router_id = ? AND user_id = ?", id, userID).Delete(&ds.ProviderRouterLike{}).Error
		}
		if err != nil {
			return err
		}
		likesCount = tx.Model(&item).Association("Likes").Count()
		return nil
	})
	return likesCount, err
}

// MediaURL — публичная ссылка на файл маршрутизатора в MinIO.
func (r *Repository) MediaURL(key, fallback string) string { return r.media.URL(key, fallback) }
