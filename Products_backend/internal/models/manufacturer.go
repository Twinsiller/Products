package models

type Manufacturer struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	Name string `gorm:"type:text;not null" json:"name"`

	Country string `gorm:"type:text" json:"country"`

	ContactInfo string `gorm:"type:jsonb" json:"contact_info"`

	// Связи
	Products []Product `gorm:"foreignKey:ManufacturerID" json:"products,omitempty"`
}

type CreateManufacturer struct {
	Name        string `json:"name" binding:"required"`
	Country     string `json:"country"`
	ContactInfo string `json:"contact_info"`
}

type UpdateManufacturer struct {
	Name        *string `json:"name"`
	Country     *string `json:"country"`
	ContactInfo *string `json:"contact_info"`
}
