package domain

import "gorm.io/gorm"

type Cart struct {
	gorm.Model
	UserID string     `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null"`
	Items  []CartItem `gorm:"foreignKey:CartID"`
}

func (Cart) TableName() string { return "carts" }

type CartItem struct {
	gorm.Model
	CartID    uint    `gorm:"column:cart_id;index;not null"`
	ProductID string  `gorm:"column:product_id;type:varchar(36);not null"`
	Quantity  int     `gorm:"column:quantity;not null"`
	Price     float64 `gorm:"column:price;type:decimal(20,8)"`
}

func (CartItem) TableName() string { return "cart_items" }

func (c *Cart) Total() float64 {
	var t float64
	for _, item := range c.Items {
		t += item.Price * float64(item.Quantity)
	}
	return t
}

func (c *Cart) AddItem(productID string, qty int, price float64) {
	for i := range c.Items {
		if c.Items[i].ProductID == productID {
			c.Items[i].Quantity += qty
			return
		}
	}
	c.Items = append(c.Items, CartItem{ProductID: productID, Quantity: qty, Price: price})
}

func (c *Cart) RemoveItem(productID string) {
	for i := range c.Items {
		if c.Items[i].ProductID == productID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			return
		}
	}
}
