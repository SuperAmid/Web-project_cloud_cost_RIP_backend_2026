package repository

import (
	"errors"

	"gorm.io/gorm"

	"provider_routers/internal/ds"
)

// CreateProviderRouterUser регистрирует пользователя; email уникален.
func (r *Repository) CreateProviderRouterUser(user *ds.ProviderRouterUser) error {
	err := r.db.Create(user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrConflict
	}
	return err
}

func (r *Repository) FindProviderRouterUserByEmail(email string) (ds.ProviderRouterUser, error) {
	var user ds.ProviderRouterUser
	err := r.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user, ErrNotFound
	}
	return user, err
}
