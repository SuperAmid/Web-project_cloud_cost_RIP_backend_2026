package repository

import (
	"sync"

	"provider_routers/internal/ds"
)

// fixedProviderRouterUserEmail — в ЛР3 авторизации нет, поэтому текущий
// пользователь-создатель зафиксирован константой.
const fixedProviderRouterUserEmail = "student@provider-routers.local"

var (
	currentProviderRouterUserMu sync.Mutex
	currentProviderRouterUser   *ds.ProviderRouterUser // единственный экземпляр
)

// CurrentProviderRouterUser — функция-singleton: при первом вызове находит
// пользователя по константе и запоминает его, дальше возвращает тот же объект
// без обращения к БД. Используется во всех методах API вместо авторизации.
// Неудачный поиск не кешируется, чтобы следующий запрос мог повторить попытку.
func (r *Repository) CurrentProviderRouterUser() (ds.ProviderRouterUser, error) {
	currentProviderRouterUserMu.Lock()
	defer currentProviderRouterUserMu.Unlock()
	if currentProviderRouterUser == nil {
		var user ds.ProviderRouterUser
		if err := r.db.Where("email = ?", fixedProviderRouterUserEmail).First(&user).Error; err != nil {
			return ds.ProviderRouterUser{}, err
		}
		currentProviderRouterUser = &user
	}
	return *currentProviderRouterUser, nil
}
