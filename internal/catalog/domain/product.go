package domain

import "gorm.io/gorm"

type Product struct {
	gorm.Model
	Name        string  `gorm:"column:name;type:varchar(255);not null"`
	Description string  `gorm:"column:description;type:text"`
	Price       float64 `gorm:"column:price;type:decimal(20,8);not null"`
	Stock       int     `gorm:"column:stock;not null;default:0"`
	Category    string  `gorm:"column:category;type:varchar(100);index"`
}

func (Product) TableName() string { return "products" }
