package component

import (
	"fmt"
	"github.com/allegro/bigcache"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"time"
)

var (
	DB          *gorm.DB
	GlobalCache *bigcache.BigCache
)

func init() {
	// Connect to DB
	var err error
	DB, err = gorm.Open("mysql", "your_db_url")
	if err != nil {
		panic(fmt.Sprintf("failed to connect to DB: %v", err))
	}

	// Initialize cache
	GlobalCache, err = bigcache.NewBigCache(bigcache.DefaultConfig(30 * time.Minute))
	if err != nil {
		panic(fmt.Sprintf("failed to initialize cahce: %v", err))
	}
}
