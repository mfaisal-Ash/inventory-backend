package docnumber

import (
	"fmt"
	"time"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Next(db *gorm.DB, prefix string) (string, error) {
	period := time.Now().Format("200601")
	key := prefix + "-" + period
	var counter model.DocumentCounter
	// PostgreSQL requires the target table qualification in this upsert when
	// referring to the existing row. SQLite (used for local development) does
	// not accept that qualification, so keep the expression dialect-specific.
	counterExpr := "counter + 1"
	if db.Dialector.Name() == "postgres" {
		counterExpr = "document_counters.counter + 1"
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"counter": gorm.Expr(counterExpr), "updated_at": time.Now()}),
	}).Create(&model.DocumentCounter{Key: key, Counter: 1, UpdatedAt: time.Now()}).Error; err != nil {
		return "", err
	}
	if err := db.Where("key = ?", key).First(&counter).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%04d", prefix, period, counter.Counter), nil
}
