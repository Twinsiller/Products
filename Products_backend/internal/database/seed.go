package database

import (
	"Products_backend/internal/models"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedIfEmpty заполняет базу демо-данными, если каталог пуст.
// Срабатывает только когда в таблице products нет ни одной записи —
// поэтому безопасно вызывать при каждом запуске.
func SeedIfEmpty(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Product{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// === Категории ===
	categories := []models.Category{
		{Name: "Молочные продукты"},      // 1
		{Name: "Мясо и птица"},           // 2
		{Name: "Овощи"},                  // 3
		{Name: "Фрукты"},                 // 4
		{Name: "Крупы и макароны"},       // 5
		{Name: "Хлеб и выпечка"},         // 6
		{Name: "Яйца"},                   // 7
	}
	if err := db.Create(&categories).Error; err != nil {
		return err
	}

	// === Производители ===
	manufacturers := []models.Manufacturer{
		{Name: "АгроФерма", Country: "Россия", ContactInfo: `{"email":"info@agroferma.ru","phone":"+7-495-100-10-10"}`},
		{Name: "МолочКом", Country: "Россия", ContactInfo: `{"email":"sales@molokom.ru","phone":"+7-495-200-20-20"}`},
		{Name: "Здоровая еда", Country: "Россия", ContactInfo: `{"email":"hello@zdorovaya-eda.ru","phone":"+7-495-300-30-30"}`},
	}
	if err := db.Create(&manufacturers).Error; err != nil {
		return err
	}

	// Удобные ссылки на ID категорий и производителей
	catMilk, catMeat, catVeg, catFruit, catGrain, catBread, catEgg :=
		categories[0].ID, categories[1].ID, categories[2].ID, categories[3].ID,
		categories[4].ID, categories[5].ID, categories[6].ID

	mFerma, mMolok, mZdor := manufacturers[0].ID, manufacturers[1].ID, manufacturers[2].ID

	// === Продукты (30 шт) ===
	// КБЖУ — на 100 г готового продукта.
	products := []models.Product{
		// Молочные
		{Name: "Молоко 3.2%, 1 л", CategoryID: &catMilk, ManufacturerID: &mMolok, DefaultPrice: 89.90, CaloriesKcal: 60, ProteinG: 3.0, FatG: 3.2, CarbsG: 4.7},
		{Name: "Кефир 2.5%, 1 л", CategoryID: &catMilk, ManufacturerID: &mMolok, DefaultPrice: 79.50, CaloriesKcal: 53, ProteinG: 2.9, FatG: 2.5, CarbsG: 4.0},
		{Name: "Творог 5%, 200 г", CategoryID: &catMilk, ManufacturerID: &mMolok, DefaultPrice: 119.00, CaloriesKcal: 121, ProteinG: 17.2, FatG: 5.0, CarbsG: 1.8},
		{Name: "Сыр Российский, 200 г", CategoryID: &catMilk, ManufacturerID: &mMolok, DefaultPrice: 199.00, CaloriesKcal: 363, ProteinG: 24.1, FatG: 29.5, CarbsG: 0.3},
		{Name: "Сметана 20%, 350 г", CategoryID: &catMilk, ManufacturerID: &mMolok, DefaultPrice: 99.00, CaloriesKcal: 206, ProteinG: 2.8, FatG: 20.0, CarbsG: 3.2},
		{Name: "Йогурт натуральный, 250 г", CategoryID: &catMilk, ManufacturerID: &mMolok, DefaultPrice: 69.90, CaloriesKcal: 60, ProteinG: 5.0, FatG: 1.5, CarbsG: 7.5},
		{Name: "Масло сливочное 82%, 180 г", CategoryID: &catMilk, ManufacturerID: &mMolok, DefaultPrice: 179.00, CaloriesKcal: 748, ProteinG: 0.5, FatG: 82.5, CarbsG: 0.8},

		// Мясо / птица
		{Name: "Куриное филе, 1 кг", CategoryID: &catMeat, ManufacturerID: &mFerma, DefaultPrice: 359.00, CaloriesKcal: 113, ProteinG: 23.6, FatG: 1.9, CarbsG: 0.4},
		{Name: "Куриные бёдра, 1 кг", CategoryID: &catMeat, ManufacturerID: &mFerma, DefaultPrice: 269.00, CaloriesKcal: 211, ProteinG: 16.8, FatG: 16.0, CarbsG: 0.0},
		{Name: "Говядина (вырезка), 500 г", CategoryID: &catMeat, ManufacturerID: &mFerma, DefaultPrice: 599.00, CaloriesKcal: 187, ProteinG: 18.9, FatG: 12.4, CarbsG: 0.0},
		{Name: "Фарш говяжий, 500 г", CategoryID: &catMeat, ManufacturerID: &mFerma, DefaultPrice: 349.00, CaloriesKcal: 254, ProteinG: 17.2, FatG: 20.0, CarbsG: 0.0},
		{Name: "Сосиски молочные, 400 г", CategoryID: &catMeat, ManufacturerID: &mFerma, DefaultPrice: 229.00, CaloriesKcal: 277, ProteinG: 11.0, FatG: 23.9, CarbsG: 1.6},

		// Овощи
		{Name: "Картофель, 1 кг", CategoryID: &catVeg, ManufacturerID: &mZdor, DefaultPrice: 39.00, CaloriesKcal: 77, ProteinG: 2.0, FatG: 0.1, CarbsG: 17.5},
		{Name: "Морковь, 1 кг", CategoryID: &catVeg, ManufacturerID: &mZdor, DefaultPrice: 49.00, CaloriesKcal: 35, ProteinG: 1.3, FatG: 0.1, CarbsG: 6.9},
		{Name: "Лук репчатый, 1 кг", CategoryID: &catVeg, ManufacturerID: &mZdor, DefaultPrice: 39.00, CaloriesKcal: 41, ProteinG: 1.4, FatG: 0.0, CarbsG: 8.2},
		{Name: "Помидоры, 1 кг", CategoryID: &catVeg, ManufacturerID: &mZdor, DefaultPrice: 159.00, CaloriesKcal: 20, ProteinG: 0.6, FatG: 0.2, CarbsG: 4.2},
		{Name: "Огурцы, 1 кг", CategoryID: &catVeg, ManufacturerID: &mZdor, DefaultPrice: 139.00, CaloriesKcal: 15, ProteinG: 0.8, FatG: 0.1, CarbsG: 2.8},
		{Name: "Капуста белокочанная, 1 кг", CategoryID: &catVeg, ManufacturerID: &mZdor, DefaultPrice: 35.00, CaloriesKcal: 28, ProteinG: 1.8, FatG: 0.1, CarbsG: 4.7},
		{Name: "Перец болгарский, 1 кг", CategoryID: &catVeg, ManufacturerID: &mZdor, DefaultPrice: 199.00, CaloriesKcal: 27, ProteinG: 1.3, FatG: 0.0, CarbsG: 5.3},

		// Фрукты
		{Name: "Яблоки, 1 кг", CategoryID: &catFruit, ManufacturerID: &mZdor, DefaultPrice: 99.00, CaloriesKcal: 47, ProteinG: 0.4, FatG: 0.4, CarbsG: 9.8},
		{Name: "Бананы, 1 кг", CategoryID: &catFruit, ManufacturerID: &mZdor, DefaultPrice: 89.00, CaloriesKcal: 89, ProteinG: 1.5, FatG: 0.5, CarbsG: 21.8},

		// Крупы / макароны
		{Name: "Гречка, 800 г", CategoryID: &catGrain, ManufacturerID: &mZdor, DefaultPrice: 109.00, CaloriesKcal: 343, ProteinG: 13.3, FatG: 3.4, CarbsG: 71.5},
		{Name: "Рис длиннозёрный, 900 г", CategoryID: &catGrain, ManufacturerID: &mZdor, DefaultPrice: 129.00, CaloriesKcal: 344, ProteinG: 7.0, FatG: 1.0, CarbsG: 77.3},
		{Name: "Овсяные хлопья, 500 г", CategoryID: &catGrain, ManufacturerID: &mZdor, DefaultPrice: 89.00, CaloriesKcal: 366, ProteinG: 11.9, FatG: 7.2, CarbsG: 69.3},
		{Name: "Макароны (спагетти), 500 г", CategoryID: &catGrain, ManufacturerID: &mZdor, DefaultPrice: 79.00, CaloriesKcal: 344, ProteinG: 10.4, FatG: 1.1, CarbsG: 71.5},

		// Хлеб
		{Name: "Хлеб пшеничный, 600 г", CategoryID: &catBread, ManufacturerID: &mZdor, DefaultPrice: 49.00, CaloriesKcal: 242, ProteinG: 8.1, FatG: 1.0, CarbsG: 48.8},
		{Name: "Хлеб ржаной, 600 г", CategoryID: &catBread, ManufacturerID: &mZdor, DefaultPrice: 55.00, CaloriesKcal: 174, ProteinG: 6.6, FatG: 1.2, CarbsG: 33.4},

		// Яйца
		{Name: "Яйца куриные C1, 10 шт", CategoryID: &catEgg, ManufacturerID: &mFerma, DefaultPrice: 119.00, CaloriesKcal: 157, ProteinG: 12.7, FatG: 10.9, CarbsG: 0.7},

		// Дополнительно
		{Name: "Подсолнечное масло, 1 л", CategoryID: nil, ManufacturerID: &mZdor, DefaultPrice: 149.00, CaloriesKcal: 899, ProteinG: 0.0, FatG: 99.9, CarbsG: 0.0},
		{Name: "Соль поваренная, 1 кг", CategoryID: nil, ManufacturerID: &mZdor, DefaultPrice: 25.00, CaloriesKcal: 0, ProteinG: 0.0, FatG: 0.0, CarbsG: 0.0},
	}
	if err := db.Create(&products).Error; err != nil {
		return err
	}

	// Удобные алиасы продуктов (по позиции в массиве)
	pMolk, pKefir, pTvor, pCheese, pSmet, pYog, pButter := products[0].ID, products[1].ID, products[2].ID, products[3].ID, products[4].ID, products[5].ID, products[6].ID
	pChickF, pChickT, pBeef, pFarsh, pSausage := products[7].ID, products[8].ID, products[9].ID, products[10].ID, products[11].ID
	pPotato, pCarrot, pOnion, pTomato, pCucumb, pCabbage, pPepper := products[12].ID, products[13].ID, products[14].ID, products[15].ID, products[16].ID, products[17].ID, products[18].ID
	pApple, pBanana := products[19].ID, products[20].ID
	pBuck, pRice, pOats, pPasta := products[21].ID, products[22].ID, products[23].ID, products[24].ID
	pBreadW, pBreadR := products[25].ID, products[26].ID
	pEggs := products[27].ID
	pOil := products[28].ID
	_ = pCarrot
	_ = pCabbage
	_ = pSmet
	_ = pBreadR
	_ = pSausage
	_ = pChickT

	// === Блюда (15 шт) с составом ===
	type dishSpec struct {
		Name        string
		Ingredients []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}
	}

	dishes := []dishSpec{
		// Завтраки
		{"Овсянка на молоке с бананом", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pOats, 1, 89.00}, {pMolk, 1, 89.90}, {pBanana, 1, 89.00},
		}},
		{"Творог со сметаной и яблоком", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pTvor, 1, 119.00}, {pSmet, 1, 99.00}, {pApple, 1, 99.00},
		}},
		{"Омлет с помидорами и сыром", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pEggs, 1, 119.00}, {pTomato, 1, 159.00}, {pCheese, 1, 199.00}, {pMolk, 1, 89.90},
		}},
		{"Тосты с маслом и сыром", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pBreadW, 1, 49.00}, {pButter, 1, 179.00}, {pCheese, 1, 199.00},
		}},
		// Обеды
		{"Куриная грудка с рисом и овощами", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pChickF, 1, 359.00}, {pRice, 1, 129.00}, {pPepper, 1, 199.00}, {pOil, 1, 149.00},
		}},
		{"Гречка с куриными бёдрами", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pBuck, 1, 109.00}, {pChickT, 1, 269.00}, {pOnion, 1, 39.00}, {pOil, 1, 149.00},
		}},
		{"Спагетти болоньезе", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pPasta, 1, 79.00}, {pFarsh, 1, 349.00}, {pTomato, 1, 159.00}, {pOnion, 1, 39.00}, {pOil, 1, 149.00},
		}},
		{"Картофельное пюре с котлетой", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pPotato, 1, 39.00}, {pButter, 1, 179.00}, {pMolk, 1, 89.90}, {pFarsh, 1, 349.00}, {pEggs, 1, 119.00},
		}},
		{"Плов с говядиной", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pBeef, 1, 599.00}, {pRice, 1, 129.00}, {pCarrot, 1, 49.00}, {pOnion, 1, 39.00}, {pOil, 1, 149.00},
		}},
		// Ужины / лёгкое
		{"Греческий салат", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pTomato, 1, 159.00}, {pCucumb, 1, 139.00}, {pPepper, 1, 199.00}, {pCheese, 1, 199.00}, {pOil, 1, 149.00},
		}},
		{"Салат из капусты с морковью", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pCabbage, 1, 35.00}, {pCarrot, 1, 49.00}, {pOil, 1, 149.00},
		}},
		{"Запечённое куриное филе с овощами", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pChickF, 1, 359.00}, {pPotato, 1, 39.00}, {pCarrot, 1, 49.00}, {pOil, 1, 149.00},
		}},
		{"Сырники", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pTvor, 2, 119.00}, {pEggs, 1, 119.00}, {pOil, 1, 149.00},
		}},
		{"Кефирный коктейль с бананом", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pKefir, 1, 79.50}, {pBanana, 1, 89.00},
		}},
		{"Бутерброд с колбасой и сыром", []struct {
			ProductID int64
			Quantity  int
			Price     float64
		}{
			{pBreadR, 1, 55.00}, {pSausage, 1, 229.00}, {pCheese, 1, 199.00}, {pButter, 1, 179.00},
		}},
	}

	// Создаём блюда и связи DishProduct
	for _, ds := range dishes {
		dish := models.Dish{Name: ds.Name}
		if err := db.Create(&dish).Error; err != nil {
			return err
		}
		for _, ing := range ds.Ingredients {
			dp := models.DishProduct{
				DishID:       dish.ID,
				ProductID:    ing.ProductID,
				Quantity:     ing.Quantity,
				PricePerUnit: ing.Price,
			}
			if err := db.Create(&dp).Error; err != nil {
				return err
			}
		}
	}

	// === Пользователи (3 шт) ===
	// Пароли в открытом виде — для удобства входа в демо.
	// admin / admin123  ·  anna / anna1234  ·  ivan / ivan1234
	type userSpec struct {
		Name     string
		Password string
		Role     string
		Gender   string
	}
	usersSpec := []userSpec{
		{"admin", "admin123", "admin", "male"},
		{"anna", "anna1234", "user", "female"},
		{"ivan", "ivan1234", "user", "male"},
	}
	createdUsers := make([]models.User, 0, len(usersSpec))
	for _, us := range usersSpec {
		hash, err := bcrypt.GenerateFromPassword([]byte(us.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u := models.User{
			Name:         us.Name,
			Role:         us.Role,
			PasswordHash: string(hash),
			Gender:       us.Gender,
			HiredAt:      time.Now(),
		}
		if err := db.Create(&u).Error; err != nil {
			return err
		}
		createdUsers = append(createdUsers, u)
	}

	uAdmin, uAnna, uIvan := createdUsers[0].ID, createdUsers[1].ID, createdUsers[2].ID
	_ = uAdmin

	// === История заказов (нужна для рекомендаций) ===
	// Anna — любит молочку и фрукты, лёгкие завтраки
	createOrder(db, uAnna, time.Now().AddDate(0, 0, -14), []struct {
		PID   int64
		Qty   int
		Price float64
	}{
		{pOats, 1, 89.00}, {pMolk, 2, 89.90}, {pBanana, 1, 89.00}, {pYog, 2, 69.90},
	})
	createOrder(db, uAnna, time.Now().AddDate(0, 0, -7), []struct {
		PID   int64
		Qty   int
		Price float64
	}{
		{pTvor, 2, 119.00}, {pSmet, 1, 99.00}, {pApple, 1, 99.00}, {pKefir, 1, 79.50},
	})
	createOrder(db, uAnna, time.Now().AddDate(0, 0, -2), []struct {
		PID   int64
		Qty   int
		Price float64
	}{
		{pCucumb, 1, 139.00}, {pTomato, 1, 159.00}, {pPepper, 1, 199.00}, {pCheese, 1, 199.00},
	})

	// Ivan — мясо, гречка, спортивное питание
	createOrder(db, uIvan, time.Now().AddDate(0, 0, -10), []struct {
		PID   int64
		Qty   int
		Price float64
	}{
		{pChickF, 2, 359.00}, {pBuck, 1, 109.00}, {pRice, 1, 129.00}, {pEggs, 2, 119.00},
	})
	createOrder(db, uIvan, time.Now().AddDate(0, 0, -5), []struct {
		PID   int64
		Qty   int
		Price float64
	}{
		{pBeef, 1, 599.00}, {pPotato, 1, 39.00}, {pCarrot, 1, 49.00}, {pOnion, 1, 39.00},
	})
	createOrder(db, uIvan, time.Now().AddDate(0, 0, -1), []struct {
		PID   int64
		Qty   int
		Price float64
	}{
		{pPasta, 1, 79.00}, {pFarsh, 1, 349.00}, {pTomato, 1, 159.00},
	})

	// === Избранное (для рекомендаций) ===
	favs := []models.FavouriteProduct{
		{UserID: uAnna, ProductID: pYog},
		{UserID: uAnna, ProductID: pTvor},
		{UserID: uAnna, ProductID: pApple},
		{UserID: uIvan, ProductID: pChickF},
		{UserID: uIvan, ProductID: pBuck},
		{UserID: uIvan, ProductID: pEggs},
	}
	if err := db.Create(&favs).Error; err != nil {
		return err
	}

	return nil
}

// createOrder — служебная утилита: создаёт заказ с позициями и считает сумму.
func createOrder(db *gorm.DB, userID int64, createdAt time.Time, items []struct {
	PID   int64
	Qty   int
	Price float64
}) {
	order := models.Order{
		UserID:    userID,
		CreatedAt: createdAt,
	}
	if err := db.Create(&order).Error; err != nil {
		return
	}
	var total float64
	for _, it := range items {
		oi := models.OrderItem{
			OrderID:      order.ID,
			ProductID:    it.PID,
			Quantity:     it.Qty,
			PricePerUnit: it.Price,
		}
		_ = db.Create(&oi).Error
		total += float64(it.Qty) * it.Price
	}
	_ = db.Model(&order).Update("total_amount", total).Error
}
