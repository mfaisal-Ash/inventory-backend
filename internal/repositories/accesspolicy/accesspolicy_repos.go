package accesspolicy

import (
	"errors"

	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
)

const singletonID = 1

func (r *repository) Get() (*model.AccessPolicy, error) {
	var p model.AccessPolicy
	err := r.db.First(&p, singletonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) SaveAccessPolicy(networks string, adminGudangOnly bool, updatedBy uint) (*model.AccessPolicy, error) {
	p := model.AccessPolicy{
		ID:               singletonID,
		KaryawanNetworks: networks,
		AdminGudangOnly:  adminGudangOnly,
		UpdatedBy:        &updatedBy,
	}
	if err := r.db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
